package gps

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config controls the GPS reader.
//
// The GPYes 2.0 (u-blox8) typically appears as /dev/ttyACM* and outputs NMEA
// (often GNxxx talker IDs) at 9600 baud by default.
//
// This package intentionally focuses on NMEA RMC+GGA which are sufficient for
// GDL90 ownship (lat/lon/gs/trk/alt).
//
// Note: This is a best-effort bring-up service; failures should not bring down
// the main process.
//
// Device may be empty to auto-detect.
// Baud must be a supported rate by the platform implementation.
//
// HorizontalAccuracyM is not used by this package; it’s consumed at a higher
// layer to derive NACp.
//
// All fields are optional unless noted.
//
//go:generate go test ./...

type Config struct {
	Enable bool

	// Source selects how GPS is ingested: "nmea" (direct serial) or "gpsd".
	// When empty, defaults to "nmea".
	Source string

	// GPSDAddr is host:port for gpsd when Source=="gpsd".
	GPSDAddr string

	// Device is the serial device path for Source=="nmea".
	Device string
	Baud   int
}

type Snapshot struct {
	Enabled  bool `json:"enabled"`
	Valid    bool `json:"valid"`
	FixStale bool `json:"fix_stale"`

	Source   string `json:"source,omitempty"`
	GPSDAddr string `json:"gpsd_addr,omitempty"`

	Device string `json:"device,omitempty"`
	Baud   int    `json:"baud,omitempty"`

	LatDeg       float64  `json:"lat_deg,omitempty"`
	LonDeg       float64  `json:"lon_deg,omitempty"`
	AltFeet      *int     `json:"alt_feet,omitempty"`
	GroundKt     *int     `json:"ground_kt,omitempty"`
	TrackDeg     *float64 `json:"track_deg,omitempty"`
	FixQuality   *int     `json:"fix_quality,omitempty"`
	FixMode      *int     `json:"fix_mode,omitempty"`
	Satellites   *int     `json:"satellites,omitempty"`
	HDOP         *float64 `json:"hdop,omitempty"`
	HorizAccM    *float64 `json:"horiz_acc_m,omitempty"`
	VertAccM     *float64 `json:"vert_acc_m,omitempty"`
	VertSpeedFPM *int     `json:"vert_speed_fpm,omitempty"`
	FixAgeSec    float64  `json:"fix_age_sec,omitempty"`

	LastFixUTC string `json:"last_fix_utc,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type Service struct {
	cfg Config

	cancel context.CancelFunc
	wg     sync.WaitGroup

	last atomic.Value // Snapshot

	mu     sync.Mutex
	closer io.Closer
}

func New(cfg Config) *Service {
	s := &Service{cfg: cfg}
	src := strings.ToLower(strings.TrimSpace(cfg.Source))
	if src == "" {
		src = "nmea"
	}
	s.last.Store(Snapshot{Enabled: cfg.Enable, Source: src, GPSDAddr: strings.TrimSpace(cfg.GPSDAddr), Device: cfg.Device, Baud: cfg.Baud})
	return s
}

func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("gps service is nil")
	}
	if !s.cfg.Enable {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("ctx is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}

	src := strings.ToLower(strings.TrimSpace(s.cfg.Source))
	if src == "" {
		src = "nmea"
	}
	if src == "gpsd" {
		return s.startGPSDLocked(ctx)
	}
	// Default: NMEA over serial.
	return s.startNMEALocked(ctx)
}

func (s *Service) startNMEALocked(ctx context.Context) error {

	device := strings.TrimSpace(s.cfg.Device)
	if device == "" {
		device = autoDetectDevice()
		if device == "" {
			s.setErrorLocked("gps auto-detect failed: no /dev/ttyACM* or /dev/ttyUSB* found")
			return fmt.Errorf("gps auto-detect failed")
		}
	}

	baud := s.cfg.Baud
	if baud == 0 {
		baud = 9600
	}

	f, err := openSerial(device, baud)
	if err != nil {
		s.setErrorLocked(fmt.Sprintf("gps open failed device=%s baud=%d: %v", device, baud, err))
		return err
	}
	// Keep the file reference for Close().
	s.closer = f

	childCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			_ = f.Close()
		}()

		log.Printf("gps enabled device=%s baud=%d", device, baud)

		reader := bufio.NewScanner(f)
		// NMEA sentences are typically < 82 chars, but allow some headroom.
		reader.Buffer(make([]byte, 0, 256), 4096)

		var st nmeaState
		st.device = device
		st.baud = baud

		for {
			select {
			case <-childCtx.Done():
				return
			default:
			}

			if !reader.Scan() {
				err := reader.Err()
				if err == nil {
					err = io.EOF
				}
				s.setError(fmt.Sprintf("gps read stopped: %v", err))
				return
			}

			line := strings.TrimSpace(reader.Text())
			if line == "" {
				continue
			}
			// Some receivers may include non-NMEA chatter; filter quickly.
			if !strings.HasPrefix(line, "$") {
				continue
			}

			sent, perr := parseNMEASentence(line)
			if perr != nil {
				// Avoid spamming on bad noise; just keep the last error.
				s.setError(perr.Error())
				continue
			}

			if updated := st.apply(time.Now().UTC(), sent); updated {
				s.last.Store(st.snapshot())
			}
		}
	}()

	// Publish initial snapshot.
	s.last.Store(Snapshot{Enabled: true, Valid: false, Source: "nmea", Device: device, Baud: baud})
	return nil
}

func (s *Service) startGPSDLocked(ctx context.Context) error {
	addr := strings.TrimSpace(s.cfg.GPSDAddr)
	if addr == "" {
		addr = "127.0.0.1:2947"
	}

	childCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		log.Printf("gps enabled source=gpsd addr=%s", addr)
		st := newGPSDState(addr)
		backoff := 250 * time.Millisecond
		maxBackoff := 10 * time.Second

		for {
			select {
			case <-childCtx.Done():
				return
			default:
			}

			conn, err := dialGPSD(childCtx, addr)
			if err != nil {
				s.setError(fmt.Sprintf("gpsd dial failed addr=%s: %v", addr, err))
				t := backoff
				if t > maxBackoff {
					t = maxBackoff
				}
				select {
				case <-childCtx.Done():
					return
				case <-time.After(t):
				}
				if backoff < maxBackoff {
					backoff *= 2
				}
				continue
			}

			// Reset backoff after a successful connection.
			backoff = 250 * time.Millisecond

			s.mu.Lock()
			// Swap the closer so Close() can interrupt an active connection.
			s.closer = conn
			s.mu.Unlock()

			func() {
				defer func() { _ = conn.Close() }()

				// Start watching JSON reports.
				if err := gpsdWatch(conn); err != nil {
					s.setError(fmt.Sprintf("gpsd watch failed: %v", err))
					return
				}

				scanner := bufio.NewScanner(conn)
				scanner.Buffer(make([]byte, 0, 4096), 256*1024)
				for {
					select {
					case <-childCtx.Done():
						return
					default:
					}
					if !scanner.Scan() {
						err := scanner.Err()
						if err == nil {
							err = io.EOF
						}
						s.setError(fmt.Sprintf("gpsd read stopped: %v", err))
						return
					}
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}
					updated, perr := st.applyLine(time.Now().UTC(), line)
					if perr != nil {
						s.setError(perr.Error())
						continue
					}
					if updated {
						s.last.Store(st.snapshot())
					}
				}
			}()
			// Loop and reconnect.
		}
	}()

	// Publish initial snapshot.
	s.last.Store(Snapshot{Enabled: true, Valid: false, Source: "gpsd", GPSDAddr: addr, Device: "gpsd"})
	return nil
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	closer := s.closer
	s.cancel = nil
	s.closer = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if closer != nil {
		_ = closer.Close()
	}
	s.wg.Wait()
}

func (s *Service) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	v := s.last.Load()
	if v == nil {
		return Snapshot{}
	}
	return v.(Snapshot)
}

func (s *Service) setError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setErrorLocked(msg)
}

func (s *Service) setErrorLocked(msg string) {
	cur := s.Snapshot()
	cur.LastError = msg
	// Do not force Valid=false here; transient parse issues shouldn’t flip validity.
	s.last.Store(cur)
}

func autoDetectDevice() string {
	// Keep it intentionally tiny and predictable.
	candidates := []string{}
	for i := 0; i < 10; i++ {
		candidates = append(candidates, fmt.Sprintf("/dev/ttyACM%d", i))
	}
	for i := 0; i < 10; i++ {
		candidates = append(candidates, fmt.Sprintf("/dev/ttyUSB%d", i))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
