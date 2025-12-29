package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/gps"
	"stratux-ng/internal/replay"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
	"stratux-ng/internal/wifi"
)

type ahrsProxy struct {
	mu sync.RWMutex
	rt *liveRuntime
}

func (p *ahrsProxy) setRuntime(rt *liveRuntime) {
	p.mu.Lock()
	p.rt = rt
	p.mu.Unlock()
}

func (p *ahrsProxy) clearRuntime(rt *liveRuntime) {
	p.mu.Lock()
	if p.rt == rt {
		p.rt = nil
	}
	p.mu.Unlock()
}

func (p *ahrsProxy) SetLevel() error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSSetLevel()
}

func (p *ahrsProxy) ZeroDrift(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSZeroDrift(ctx)
}

func (p *ahrsProxy) OrientForward(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSOrientForward(ctx)
}

func (p *ahrsProxy) OrientDone(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSOrientDone(ctx)
}

func (p *ahrsProxy) Orientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return 0, [3]float64{}, false
	}
	return rt.AHRSOrientation()
}

func decodeAttitudeFromFrames(frames [][]byte) web.AttitudeSnapshot {
	var out web.AttitudeSnapshot

	for _, frame := range frames {
		msg, crcOK, err := gdl90.Unframe(frame)
		if err != nil || !crcOK || len(msg) == 0 {
			continue
		}

		// Stratux heartbeat: bit0 = AHRS valid.
		if msg[0] == 0xCC && len(msg) >= 2 {
			out.Valid = (msg[1] & 0x01) != 0
		}

		// ForeFlight AHRS: 0x65, sub-id 0x01.
		if msg[0] == 0x65 && len(msg) >= 12 && msg[1] == 0x01 {
			roll := int16(msg[2])<<8 | int16(msg[3])
			pitch := int16(msg[4])<<8 | int16(msg[5])
			if roll != int16(0x7FFF) {
				v := float64(roll) / 10.0
				out.RollDeg = &v
			}
			if pitch != int16(0x7FFF) {
				v := float64(pitch) / 10.0
				out.PitchDeg = &v
			}
			continue
		}

		// Stratux LE AHRS report: starts with 0x4C 0x45 0x01 0x01.
		if len(msg) >= 24 && msg[0] == 0x4C && msg[1] == 0x45 && msg[2] == 0x01 && msg[3] == 0x01 {
			roll := int16(msg[4])<<8 | int16(msg[5])
			pitch := int16(msg[6])<<8 | int16(msg[7])
			hdg := int16(msg[8])<<8 | int16(msg[9])
			palt := uint16(msg[18])<<8 | uint16(msg[19])
			if roll != int16(0x7FFF) {
				v := float64(roll) / 10.0
				out.RollDeg = &v
			}
			if pitch != int16(0x7FFF) {
				v := float64(pitch) / 10.0
				out.PitchDeg = &v
			}
			if hdg != int16(0x7FFF) {
				v := float64(hdg) / 10.0
				out.HeadingDeg = &v
			}
			if palt != uint16(0xFFFF) {
				v := float64(palt) - 5000.5
				out.PressureAltFt = &v
			}
			continue
		}
	}
	return out
}

func attitudeSnapshotFromAHRS(snap ahrs.Snapshot) web.AttitudeSnapshot {
	att := web.AttitudeSnapshot{}
	attValid := snap.Valid
	if snap.IMUDetected && !snap.StartupReady {
		attValid = false
	}
	att.Valid = attValid
	if attValid {
		roll := snap.RollDeg
		pitch := snap.PitchDeg
		att.RollDeg = &roll
		att.PitchDeg = &pitch
		if snap.PressureAltValid {
			v := snap.PressureAltFeet
			att.PressureAltFt = &v
		}
		if snap.StartupReady && snap.GLoadValid {
			g := snap.GLoadG
			gmin := snap.GLoadMinG
			gmax := snap.GLoadMaxG
			att.GLoad = &g
			att.GMin = &gmin
			att.GMax = &gmax
		}
	}
	if !snap.UpdatedAt.IsZero() {
		att.LastUpdateUTC = snap.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return att
}

func decodeTrafficFromFrames(frames [][]byte) []web.TrafficSnapshot {
	// Keep the decode logic local to main for now. We only need enough fields
	// to render targets on the web map.
	const latLonResolution = 180.0 / 8388608.0
	const trackResolution = 360.0 / 256.0

	trafficByICAO := map[string]web.TrafficSnapshot{}
	order := make([]string, 0, 16)

	decodeSigned24 := func(b0, b1, b2 byte) int32 {
		u := int32(b0)<<16 | int32(b1)<<8 | int32(b2)
		// Sign-extend 24-bit.
		if (u & 0x00800000) != 0 {
			u |= ^int32(0x00FFFFFF)
		}
		return u
	}

	decodeSigned12 := func(v uint16) int16 {
		x := int16(v & 0x0FFF)
		if (x & 0x0800) != 0 {
			x |= ^int16(0x0FFF)
		}
		return x
	}

	for _, frame := range frames {
		msg, crcOK, err := gdl90.Unframe(frame)
		if err != nil || !crcOK || len(msg) < 28 {
			continue
		}
		if msg[0] != 0x14 {
			continue
		}

		icao := fmt.Sprintf("%02X%02X%02X", msg[2], msg[3], msg[4])
		if _, ok := trafficByICAO[icao]; !ok {
			order = append(order, icao)
		}

		lat24 := decodeSigned24(msg[5], msg[6], msg[7])
		lon24 := decodeSigned24(msg[8], msg[9], msg[10])
		lat := float64(lat24) * latLonResolution
		lon := float64(lon24) * latLonResolution

		alt12 := (uint16(msg[11]) << 4) | (uint16(msg[12]) >> 4)
		altFeet := 0
		if alt12 != 0x0FFF {
			altFeet = int(alt12)*25 - 1000
		}

		flags := msg[12] & 0x0F
		extrap := (flags & 0x04) != 0
		onGround := (flags & 0x08) == 0

		spd12 := (uint16(msg[14]) << 4) | (uint16(msg[15]) >> 4)
		groundKt := int(spd12 & 0x0FFF)

		vv12 := (uint16(msg[15]&0x0F) << 8) | uint16(msg[16])
		vvelFpm := int(decodeSigned12(vv12)) * 64

		trk := float64(msg[17]) * trackResolution

		tail := strings.TrimRight(string(msg[19:27]), " ")

		trafficByICAO[icao] = web.TrafficSnapshot{
			ICAO:         icao,
			Tail:         tail,
			LatDeg:       lat,
			LonDeg:       lon,
			AltFeet:      altFeet,
			GroundKt:     groundKt,
			TrackDeg:     trk,
			VvelFpm:      vvelFpm,
			OnGround:     onGround,
			Extrapolated: extrap,
		}
	}

	// Stable output for UI.
	sort.Strings(order)
	out := make([]web.TrafficSnapshot, 0, len(order))
	for _, icao := range order {
		out = append(out, trafficByICAO[icao])
	}
	return out
}

type ctxSleeper struct{ ctx context.Context }

func (cs ctxSleeper) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-cs.ctx.Done():
		return
	case <-t.C:
		return
	}
}

type replayOpener func(path string) (io.ReadCloser, error)
type frameSender func(frame []byte) error

type applyRequest struct {
	cfg  config.Config
	resp chan error
}

type safeBroadcaster struct {
	mu sync.Mutex
	b  *udp.Broadcaster
}

func (s *safeBroadcaster) Send(payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.b
	if b == nil {
		return errors.New("udp broadcaster is nil")
	}
	return b.Send(payload)
}

func (s *safeBroadcaster) Swap(next *udp.Broadcaster) {
	s.mu.Lock()
	old := s.b
	s.b = next
	if old != nil {
		_ = old.Close()
	}
	s.mu.Unlock()
}

func (s *safeBroadcaster) Close() {
	s.mu.Lock()
	old := s.b
	s.b = nil
	if old != nil {
		_ = old.Close()
	}
	s.mu.Unlock()
}

func staticInfoSnapshot(resolvedConfigPath string, cfg config.Config) map[string]any {
	return map[string]any{
		"config_path":      resolvedConfigPath,
		"record":           cfg.GDL90.Record.Enable,
		"replay":           cfg.GDL90.Replay.Enable,
		"ownship_icao":     cfg.Ownship.ICAO,
		"ownship_callsign": cfg.Ownship.Callsign,
	}
}

func runReplay(ctx context.Context, cfg config.Config, open replayOpener, send frameSender) error {
	if open == nil {
		open = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	if send == nil {
		return errors.New("send is nil")
	}

	rc, err := open(cfg.GDL90.Replay.Path)
	if err != nil {
		return err
	}
	defer rc.Close()

	recs, err := replay.NewReader(rc).ReadAll()
	if err != nil {
		return err
	}

	return replay.Play(recs, cfg.GDL90.Replay.Speed, cfg.GDL90.Replay.Loop, ctxSleeper{ctx: ctx}, func(frame []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return send(frame)
	})
}

func main() {
	var configPath string
	var resolvedConfigPath string
	var recordPath string
	var replayPath string
	var replaySpeed float64
	var replayLoop bool
	var logSummaryPath string
	var listenMode bool
	var listenAddr string
	var listenHex bool
	var webListen string

	flag.StringVar(&configPath, "config", "", "Path to YAML config (optional; defaults to /data/stratux-ng/config.yaml; STRATUX_NG_CONFIG overrides)")
	flag.StringVar(&recordPath, "record", "", "Record framed GDL90 packets to PATH (overrides config)")
	flag.StringVar(&replayPath, "replay", "", "Replay framed GDL90 packets from PATH (overrides config)")
	flag.Float64Var(&replaySpeed, "replay-speed", -1, "Replay speed multiplier (e.g., 2.0 = 2x). -1 uses config")
	flag.BoolVar(&replayLoop, "replay-loop", false, "Loop replay forever (overrides config when true)")
	flag.StringVar(&logSummaryPath, "log-summary", "", "Print summary of a record/replay log at PATH and exit")
	flag.BoolVar(&listenMode, "listen", false, "Listen for UDP GDL90 frames and dump decoded messages (no transmit)")
	flag.StringVar(&listenAddr, "listen-addr", ":4000", "UDP address to bind in listen mode (e.g. :4000 or 127.0.0.1:4000)")
	flag.BoolVar(&listenHex, "listen-hex", false, "In listen mode, also print raw frame bytes as hex")
	flag.StringVar(&webListen, "web-listen", "", "Web UI listen address (overrides config when non-empty)")
	flag.Parse()

	if strings.TrimSpace(logSummaryPath) != "" {
		if err := printLogSummary(logSummaryPath); err != nil {
			log.Fatalf("log summary failed: %v", err)
		}
		return
	}
	if listenMode {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		if err := runListen(ctx, listenAddr, listenHex); err != nil && ctx.Err() == nil {
			log.Fatalf("listen mode failed: %v", err)
		}
		return
	}

	cfg, resolved, err := config.LoadAuto(configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	resolvedConfigPath = resolved

	// Initialize Wi-Fi AP if running as root.
	if os.Geteuid() == 0 {
		if err := wifi.EnsureAPInterface(); err != nil {
			log.Printf("wifi: failed to ensure AP interface: %v", err)
		} else {
			// Use configured SSID or default to "stratux".
			ssid := cfg.WiFi.APSSID
			if ssid == "" {
				ssid = "stratux"
			}
			if err := wifi.SetupAP(ssid, cfg.WiFi.APPass, cfg.WiFi.APIP); err != nil {
				log.Printf("wifi: failed to setup AP: %v", err)
			} else {
				log.Printf("wifi: AP %q configured", ssid)
			}

			// If client is configured, try to connect.
			if cfg.WiFi.ClientSSID != "" {
				if err := wifi.ConnectClient(cfg.WiFi.ClientSSID, cfg.WiFi.ClientPass); err != nil {
					log.Printf("wifi: failed to connect client: %v", err)
				} else {
					log.Printf("wifi: client connecting to %q", cfg.WiFi.ClientSSID)
				}
			}
		}
	}

	// CLI overrides.
	if strings.TrimSpace(webListen) != "" {
		cfg.Web.Listen = webListen
	}

	// CLI overrides (useful for quick record/replay without editing YAML).
	if recordPath != "" {
		cfg.GDL90.Record.Enable = true
		cfg.GDL90.Record.Path = recordPath
	}
	if replayPath != "" {
		cfg.GDL90.Replay.Enable = true
		cfg.GDL90.Replay.Path = replayPath
	}
	if replaySpeed >= 0 {
		cfg.GDL90.Replay.Speed = replaySpeed
	}
	if replayLoop {
		cfg.GDL90.Replay.Loop = true
	}
	if recordPath != "" || replayPath != "" || replaySpeed >= 0 || replayLoop || strings.TrimSpace(webListen) != "" {
		if err := config.DefaultAndValidate(&cfg); err != nil {
			log.Fatalf("config validation failed after CLI overrides: %v", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logBuf := web.NewLogBuffer(4000)
	log.SetOutput(io.MultiWriter(os.Stdout, logBuf))

	status := web.NewStatus()
	status.SetStatic(cfg.GDL90.Dest, cfg.GDL90.Interval.String(), staticInfoSnapshot(resolvedConfigPath, cfg))
	attitudeBroadcaster := web.NewAttitudeBroadcaster()

	applyCh := make(chan applyRequest)
	applyFunc := func(nextCfg config.Config) error {
		req := applyRequest{cfg: nextCfg, resp: make(chan error, 1)}
		select {
		case applyCh <- req:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case err := <-req.resp:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	log.Printf("web ui enabled listen=%s", cfg.Web.Listen)
	proxy := &ahrsProxy{}
	go func() {
		for {
			err := web.Serve(ctx, cfg.Web.Listen, status, web.SettingsStore{ConfigPath: resolvedConfigPath, Apply: applyFunc}, logBuf, proxy, attitudeBroadcaster)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if errors.Is(err, syscall.EACCES) {
					log.Printf("web ui bind failed (permission denied) listen=%s: %v", cfg.Web.Listen, err)
					log.Printf("to use port 80 without running as root, grant CAP_NET_BIND_SERVICE (examples):")
					log.Printf("  setcap: sudo setcap 'cap_net_bind_service=+ep' $(readlink -f ./stratux-ng)")
					log.Printf("  systemd: set AmbientCapabilities=CAP_NET_BIND_SERVICE in the service unit")
					cancel()
					return
				}
				log.Printf("web ui stopped: %v; restarting in 1s", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}
			return
		}
	}()

	var rec *replay.Writer
	if cfg.GDL90.Record.Enable {
		w, err := replay.CreateWriter(cfg.GDL90.Record.Path)
		if err != nil {
			log.Fatalf("record init failed: %v", err)
		}
		rec = w
		defer func() {
			if err := rec.Close(); err != nil {
				log.Printf("record close failed: %v", err)
			}
		}()
		log.Printf("recording enabled path=%s", cfg.GDL90.Record.Path)
	}

	initialBroadcaster, err := udp.NewBroadcaster(cfg.GDL90.Dest)
	if err != nil {
		log.Fatalf("udp broadcaster init failed: %v", err)
	}
	sender := &safeBroadcaster{b: initialBroadcaster}
	defer sender.Close()

	log.Printf("stratux-ng starting")
	log.Printf("udp dest=%s interval=%s", cfg.GDL90.Dest, cfg.GDL90.Interval)
	log.Printf("ownship icao=%s callsign=%s", cfg.Ownship.ICAO, cfg.Ownship.Callsign)

	go func() {
		rt, err := newLiveRuntime(ctx, cfg, resolvedConfigPath, status, sender, attitudeBroadcaster)
		if err != nil {
			log.Printf("runtime init failed: %v", err)
			cancel()
			return
		}
		proxy.setRuntime(rt)
		defer rt.Close()
		defer proxy.clearRuntime(rt)

		// Replay sender runs separately so we can still accept live updates.
		if rt.Config().GDL90.Replay.Enable {
			cur := rt.Config()
			log.Printf("replay enabled path=%s speed=%.3gx loop=%t", cur.GDL90.Replay.Path, cur.GDL90.Replay.Speed, cur.GDL90.Replay.Loop)
			go func() {
				err := runReplay(ctx, cur, nil, sender.Send)
				if err != nil && ctx.Err() == nil {
					log.Printf("replay stopped: %v", err)
					cancel()
				}
			}()
		}

		hf := &headingFuser{}
		var lastUDPErrorLog time.Time
		for {
			tickC := rt.TickChan()
			select {
			case <-ctx.Done():
				return
			case req := <-applyCh:
				err := rt.Apply(req.cfg)
				req.resp <- err
			case <-tickC:
				curCfg := rt.Config()
				if curCfg.GDL90.Replay.Enable {
					// Replay mode doesn't use the tick loop.
					continue
				}
				var frames [][]byte
				var snap ahrs.Snapshot
				var haveAHRS bool
				now := time.Now()
				if ds, ok := rt.ADSB1090DecoderSnapshot(now.UTC()); ok {
					status.SetADSB1090Decoder(now.UTC(), ds)
				}
				if ds, ok := rt.UAT978DecoderSnapshot(now.UTC()); ok {
					status.SetUAT978Decoder(now.UTC(), ds)
				}
				var gpsSnap gps.Snapshot
				var haveGPS bool
				if fanSnap, haveFan := rt.FanSnapshot(); haveFan {
					status.SetFan(now.UTC(), fanSnap)
				}
				if curCfg.GPS.Enable {
					gpsSnap, haveGPS = rt.GPSSnapshot()
					if haveGPS {
						if gpsSnap.LastFixUTC != "" {
							if tFix, perr := time.Parse(time.RFC3339Nano, gpsSnap.LastFixUTC); perr == nil {
								age := now.UTC().Sub(tFix.UTC()).Seconds()
								if age < 0 {
									age = 0
								}
								gpsSnap.FixAgeSec = age
								gpsSnap.FixStale = age > 3.0
							}
						}
						status.SetGPS(now.UTC(), gpsSnap)
					} else {
						status.SetGPS(now.UTC(), gps.Snapshot{Enabled: true, Valid: false})
					}
				} else {
					status.SetGPS(now.UTC(), gps.Snapshot{Enabled: false})
				}
				if curCfg.AHRS.Enable {
					snap, haveAHRS = rt.AHRSSnapshot()
					// Publish AHRS sensor health for the Status page.
					nowUTC := now.UTC()
					imuWorking := haveAHRS && snap.IMUDetected && !snap.IMULastUpdateAt.IsZero() && nowUTC.Sub(snap.IMULastUpdateAt.UTC()) <= 2*time.Second
					baroWorking := haveAHRS && snap.BaroDetected && !snap.BaroLastUpdateAt.IsZero() && nowUTC.Sub(snap.BaroLastUpdateAt.UTC()) <= 5*time.Second
					ah := web.AHRSSensorsSnapshot{
						Enabled:        true,
						IMUDetected:    haveAHRS && snap.IMUDetected,
						BaroDetected:   haveAHRS && snap.BaroDetected,
						IMUWorking:     imuWorking,
						BaroWorking:    baroWorking,
						OrientationSet: snap.OrientationSet,
						ForwardAxis:    snap.OrientationForwardAxis,
						LastError:      snap.LastError,
					}
					if !snap.IMULastUpdateAt.IsZero() {
						ah.IMULastUpdateUTC = snap.IMULastUpdateAt.UTC().Format(time.RFC3339Nano)
					}
					if !snap.BaroLastUpdateAt.IsZero() {
						ah.BaroLastUpdateUTC = snap.BaroLastUpdateAt.UTC().Format(time.RFC3339Nano)
					}
					status.SetAHRSSensors(nowUTC, ah)
				} else {
					status.SetAHRSSensors(now.UTC(), web.AHRSSensorsSnapshot{Enabled: false})
				}
				if curCfg.GPS.Enable {
					frames = buildGDL90FramesWithGPS(curCfg, now.UTC(), haveAHRS, snap, haveGPS, gpsSnap, hf, rt.TrafficTargets(now.UTC()))
				} else {
					frames = buildGDL90FramesNoGPS(curCfg, now.UTC(), haveAHRS, snap)
				}
				if extra := rt.DrainUAT978UplinkFrames(50); len(extra) > 0 {
					frames = append(frames, extra...)
				}
				attFromFrames := decodeAttitudeFromFrames(frames)
				att := attFromFrames
				if haveAHRS {
					att = attitudeSnapshotFromAHRS(snap)
					if attFromFrames.HeadingDeg != nil && att.HeadingDeg == nil {
						h := *attFromFrames.HeadingDeg
						att.HeadingDeg = &h
					}
					if attFromFrames.PressureAltFt != nil && att.PressureAltFt == nil {
						v := *attFromFrames.PressureAltFt
						att.PressureAltFt = &v
					}
				}
				status.SetTraffic(now.UTC(), decodeTrafficFromFrames(frames))
				status.SetAttitude(now.UTC(), att)
				if attitudeBroadcaster != nil {
					if att.HeadingDeg != nil {
						attitudeBroadcaster.SetHeading(*att.HeadingDeg, true)
					} else {
						attitudeBroadcaster.SetHeading(0, false)
					}
				}
				// Always record a "tick" time even if we fail mid-send.
				status.MarkTick(now.UTC(), 0)
				sent := 0
				for _, frame := range frames {
					if rec != nil {
						if err := rec.WriteFrame(now, frame); err != nil {
							log.Printf("record write failed: %v", err)
							cancel()
							return
						}
					}
					if err := sender.Send(frame); err != nil {
						if time.Since(lastUDPErrorLog) > 5*time.Second {
							log.Printf("udp send failed: %v", err)
							lastUDPErrorLog = time.Now()
						}
						// Do not crash on transient network errors.
					}
					sent++
				}
				if rec != nil {
					if err := rec.Flush(); err != nil {
						log.Printf("record flush failed: %v", err)
						cancel()
						return
					}
				}
				status.MarkTick(now.UTC(), sent)
			}
		}
	}()

	<-ctx.Done()
	log.Printf("stratux-ng stopping")
}

func runListen(ctx context.Context, addr string, dumpHex bool) error {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()

	log.Printf("listen mode: udp bind=%s", addr)
	buf := make([]byte, 64*1024)
	for {
		_ = pc.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, src, err := pc.ReadFrom(buf)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		frame := append([]byte(nil), buf[:n]...)

		msg, crcOK, uerr := gdl90.Unframe(frame)
		if uerr != nil {
			log.Printf("rx src=%s bytes=%d unframe_err=%v", src.String(), len(frame), uerr)
			if dumpHex {
				log.Printf("rx hex=%s", hex.EncodeToString(frame))
			}
			continue
		}

		id := msg[0]
		info := fmt.Sprintf("id=0x%02X", id)
		if id == 0x65 && len(msg) >= 2 {
			info = fmt.Sprintf("id=0x65 sub=0x%02X", msg[1])
		}

		log.Printf("rx src=%s bytes=%d crc_ok=%t %s msg_len=%d", src.String(), len(frame), crcOK, info, len(msg))
		if dumpHex {
			log.Printf("rx hex=%s", hex.EncodeToString(frame))
		}
	}
}

func buildGDL90FramesNoGPS(cfg config.Config, now time.Time, haveAHRS bool, ahrsSnap ahrs.Snapshot) [][]byte {
	gpsValid := false
	ahrsValid := true
	if cfg.AHRS.Enable {
		ahrsValid = haveAHRS && ahrsSnap.Valid
	}

	frames := make([][]byte, 0, 8)
	frames = append(frames,
		gdl90.HeartbeatFrameAt(now, gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, ahrsValid),
	)
	frames = append(frames, gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"))

	if !cfg.AHRS.Enable || !haveAHRS {
		return frames
	}

	pressureAltFeet := 0.0
	if ahrsSnap.PressureAltValid {
		pressureAltFeet = ahrsSnap.PressureAltFeet
	}
	vs := 0
	if ahrsSnap.VerticalSpeedValid {
		vs = ahrsSnap.VerticalSpeedFpm
	}

	att := gdl90.Attitude{
		Valid:                ahrsValid,
		RollDeg:              ahrsSnap.RollDeg,
		PitchDeg:             ahrsSnap.PitchDeg,
		HeadingDeg:           0,
		SlipSkidDeg:          0,
		YawRateDps:           ahrsSnap.YawRateDps,
		GLoad:                1.0,
		IndicatedAirspeedKt:  0,
		TrueAirspeedKt:       0,
		PressureAltitudeFeet: pressureAltFeet,
		PressureAltValid:     ahrsSnap.PressureAltValid,
		VerticalSpeedFpm:     vs,
		VerticalSpeedValid:   ahrsSnap.VerticalSpeedValid,
	}
	if ahrsSnap.GLoadValid {
		att.GLoad = ahrsSnap.GLoadG
	}
	frames = append(frames,
		gdl90.ForeFlightAHRSFrame(att),
		gdl90.AHRSGDL90LEFrame(att),
	)

	return frames
}

type headingFuser struct {
	have    bool
	heading float64
	lastAt  time.Time
}

func (h *headingFuser) Reset() {
	h.have = false
	h.heading = 0
	h.lastAt = time.Time{}
}

func (h *headingFuser) Update(now time.Time, gpsTrackDeg *float64, gpsTrackValid bool, groundKt int, yawRateDps *float64) float64 {
	// Initialize from GPS track when available.
	if !h.have {
		if gpsTrackDeg != nil {
			h.have = true
			h.heading = wrap360(*gpsTrackDeg)
			h.lastAt = now
			return h.heading
		}
		h.lastAt = now
		return 0
	}

	// If time jumped (or we were paused), re-seed from GPS if we can.
	if h.lastAt.IsZero() || now.Before(h.lastAt) || now.Sub(h.lastAt) > 2*time.Second {
		h.have = false
		return h.Update(now, gpsTrackDeg, gpsTrackValid, groundKt, yawRateDps)
	}

	dt := now.Sub(h.lastAt).Seconds()
	h.lastAt = now
	if dt <= 0 || dt > 1.0 {
		return h.heading
	}

	// Short-term: integrate yaw rate.
	if yawRateDps != nil {
		h.heading = wrap360(h.heading + (*yawRateDps)*dt)
	}

	// Long-term: converge slowly to GPS track for accuracy.
	// Gate on GPS track validity (fresh + meaningful ground speed) because COG gets noisy.
	if gpsTrackValid && gpsTrackDeg != nil {
		tau := 3.0 // seconds (larger = trust yaw more, smaller = hug GPS more)
		k := 1 - math.Exp(-dt/tau)
		diff := shortestAngleDiffDeg(*gpsTrackDeg, h.heading)
		h.heading = wrap360(h.heading + diff*k)
	}

	return h.heading
}

func wrap360(deg float64) float64 {
	deg = math.Mod(deg, 360)
	if deg < 0 {
		deg += 360
	}
	return deg
}

func shortestAngleDiffDeg(targetDeg float64, currentDeg float64) float64 {
	// Return signed delta in [-180, 180].
	d := wrap360(targetDeg) - wrap360(currentDeg)
	if d > 180 {
		d -= 360
	}
	if d < -180 {
		d += 360
	}
	return d
}

func buildGDL90FramesWithGPS(cfg config.Config, now time.Time, haveAHRS bool, ahrsSnap ahrs.Snapshot, haveGPS bool, gpsSnap gps.Snapshot, hf *headingFuser, liveTraffic []gdl90.Traffic) [][]byte {
	// GPS mode: emit ownship from live GPS when we have a recent fix.
	icao, err := gdl90.ParseICAOHex(cfg.Ownship.ICAO)
	ownshipOK := err == nil
	if err != nil {
		log.Printf("ownship invalid config ownship.icao: %v", err)
	}

	ahrsValid := true
	if cfg.AHRS.Enable {
		ahrsValid = haveAHRS && ahrsSnap.Valid
	}

	// Determine if we have a reasonably fresh fix.
	fixOK := haveGPS && gpsSnap.Enabled && gpsSnap.Valid
	if fixOK && gpsSnap.LastFixUTC != "" {
		if tFix, perr := time.Parse(time.RFC3339Nano, gpsSnap.LastFixUTC); perr == nil {
			if now.UTC().Sub(tFix.UTC()) > 3*time.Second {
				fixOK = false
			}
		}
	}

	gpsValid := ownshipOK && fixOK

	frames := make([][]byte, 0, 16)
	frames = append(frames,
		gdl90.HeartbeatFrameAt(now, gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, ahrsValid),
	)

	// Identify as a Stratux-like device for apps that key off 0x65.
	frames = append(frames, gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"))
	if !gpsValid {
		if hf != nil {
			hf.Reset()
		}
		return frames
	}

	nacp := gdl90.NACpFromHorizontalAccuracyMeters(cfg.GPS.HorizontalAccuracyM)

	geoAltFeet := 0
	if gpsSnap.AltFeet != nil {
		geoAltFeet = *gpsSnap.AltFeet
	}

	// GDL90 Ownship Report (0x0A) altitude is pressure altitude when available.
	// Mirror upstream Stratux behavior by preferring baro-derived pressure altitude.
	ownshipAltFeet := geoAltFeet
	if cfg.AHRS.Enable && haveAHRS && ahrsSnap.PressureAltValid {
		ownshipAltFeet = int(ahrsSnap.PressureAltFeet)
	}

	groundKt := 0
	if gpsSnap.GroundKt != nil {
		groundKt = *gpsSnap.GroundKt
	}
	trackDeg := 0.0
	if gpsSnap.TrackDeg != nil {
		trackDeg = *gpsSnap.TrackDeg
	}
	// GPS track is only considered valid for correction when moving fast enough.
	gpsTrackValid := fixOK && gpsSnap.TrackDeg != nil && gpsSnap.GroundKt != nil && *gpsSnap.GroundKt >= 5
	// When GPS fix quality/mode is available, require at least a 2D fix.
	if gpsTrackValid {
		if gpsSnap.FixMode != nil && *gpsSnap.FixMode < 2 {
			gpsTrackValid = false
		}
		if gpsSnap.FixQuality != nil && *gpsSnap.FixQuality <= 0 {
			gpsTrackValid = false
		}
	}

	// Fuse heading for EFB: yaw-rate for short turns, GPS track for long-term accuracy.
	// Only do this when we actually have AHRS (gyro). Otherwise use GPS track.
	headingDeg := trackDeg
	if hf != nil && cfg.AHRS.Enable && haveAHRS && ahrsSnap.Valid {
		trkPtr := (*float64)(nil)
		if gpsSnap.TrackDeg != nil {
			trkPtr = gpsSnap.TrackDeg
		}
		yawPtr := &ahrsSnap.YawRateDps
		headingDeg = hf.Update(now, trkPtr, gpsTrackValid, groundKt, yawPtr)
	}

	vvelFpm := 0
	vvelValid := false
	if gpsSnap.VertSpeedFPM != nil {
		vvelFpm = *gpsSnap.VertSpeedFPM
		vvelValid = true
	}

	frames = append(frames, gdl90.OwnshipReportFrame(gdl90.Ownship{
		ICAO:        icao,
		LatDeg:      gpsSnap.LatDeg,
		LonDeg:      gpsSnap.LonDeg,
		AltFeet:     ownshipAltFeet,
		HaveNICNACp: true,
		NIC:         8,
		NACp:        nacp,
		GroundKt:    groundKt,
		TrackDeg:    trackDeg,
		OnGround:    groundKt == 0,
		VvelFpm:     vvelFpm,
		VvelValid:   vvelValid,
		Callsign:    cfg.Ownship.Callsign,
		Emitter:     0x01,
	}))
	frames = append(frames, gdl90.OwnshipGeometricAltitudeFrame(geoAltFeet))

	// Stratux-like AHRS messages. If AHRS is present, use it; otherwise keep
	// the attitude neutral but with heading aligned to track.
	roll := 0.0
	pitch := 0.0
	if cfg.AHRS.Enable && haveAHRS {
		roll = ahrsSnap.RollDeg
		pitch = ahrsSnap.PitchDeg
	}

	pressureAltFeet := float64(geoAltFeet)
	pressureAltValid := false
	if cfg.AHRS.Enable {
		pressureAltValid = haveAHRS && ahrsSnap.PressureAltValid
		if pressureAltValid {
			pressureAltFeet = ahrsSnap.PressureAltFeet
		}
	}

	att := gdl90.Attitude{
		Valid:                ahrsValid,
		RollDeg:              roll,
		PitchDeg:             pitch,
		HeadingDeg:           headingDeg,
		SlipSkidDeg:          0,
		YawRateDps:           0,
		GLoad:                1.0,
		IndicatedAirspeedKt:  groundKt,
		TrueAirspeedKt:       groundKt,
		PressureAltitudeFeet: pressureAltFeet,
		PressureAltValid:     pressureAltValid,
		VerticalSpeedFpm:     vvelFpm,
		VerticalSpeedValid:   vvelValid,
	}
	if cfg.AHRS.Enable && haveAHRS && ahrsSnap.Valid {
		att.YawRateDps = ahrsSnap.YawRateDps
	}
	if cfg.AHRS.Enable && haveAHRS && ahrsSnap.VerticalSpeedValid {
		att.VerticalSpeedFpm = ahrsSnap.VerticalSpeedFpm
		att.VerticalSpeedValid = true
	}
	frames = append(frames,
		gdl90.ForeFlightAHRSFrame(att),
		gdl90.AHRSGDL90LEFrame(att),
	)

	for _, t := range liveTraffic {
		frames = append(frames, gdl90.TrafficReportFrame(t))
	}

	return frames
}
