package replay

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Log format: line-oriented text.
//
// - Blank lines ignored.
// - Lines starting with '#' ignored.
// - Line "START" resets the origin (next record time is relative to 0 again).
// - Data lines are: <t_ns>,<hex>
//   where t_ns is nanoseconds since START (monotonic), and hex is the raw GDL90 frame bytes.
//
// This is intentionally simple and stable for deterministic protocol regression tests.

type Record struct {
	At    time.Duration
	Frame []byte
}

type Reader struct {
	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (rr *Reader) ReadAll() ([]Record, error) {
	s := bufio.NewScanner(rr.r)
	// Allow reasonably large frames.
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	recs := make([]Record, 0, 1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line == "START" {
			recs = append(recs, Record{At: 0, Frame: nil})
			continue
		}

		comma := strings.IndexByte(line, ',')
		if comma < 0 {
			return nil, fmt.Errorf("invalid replay line (missing comma): %q", line)
		}
		tsStr := strings.TrimSpace(line[:comma])
		hexStr := strings.TrimSpace(line[comma+1:])
		if tsStr == "" || hexStr == "" {
			return nil, fmt.Errorf("invalid replay line (empty field): %q", line)
		}

		tsNs, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid replay timestamp %q: %w", tsStr, err)
		}
		if tsNs < 0 {
			return nil, fmt.Errorf("invalid replay timestamp (negative): %d", tsNs)
		}

		hexStr = strings.ReplaceAll(hexStr, " ", "")
		b, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid replay hex payload: %w", err)
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("invalid replay payload (empty)")
		}

		recs = append(recs, Record{At: time.Duration(tsNs) * time.Nanosecond, Frame: b})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return recs, nil
}

type Writer struct {
	f      *os.File
	w      *bufio.Writer
	start  time.Time
	closed bool
}

func CreateWriter(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriterSize(f, 64*1024)
	if _, err := bw.WriteString("START\n"); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Writer{f: f, w: bw, start: time.Now()}, nil
}

func (ww *Writer) WriteFrame(now time.Time, frame []byte) error {
	if ww.closed {
		return errors.New("replay writer is closed")
	}
	if frame == nil {
		return errors.New("frame is nil")
	}

	// Use monotonic component of time when available.
	d := now.Sub(ww.start)
	if d < 0 {
		d = 0
	}
	if _, err := fmt.Fprintf(ww.w, "%d,%s\n", d.Nanoseconds(), hex.EncodeToString(frame)); err != nil {
		return err
	}
	return nil
}

func (ww *Writer) Flush() error {
	if ww.closed {
		return nil
	}
	return ww.w.Flush()
}

func (ww *Writer) Close() error {
	if ww.closed {
		return nil
	}
	ww.closed = true
	if err := ww.w.Flush(); err != nil {
		_ = ww.f.Close()
		return err
	}
	return ww.f.Close()
}

type Sleeper interface {
	Sleep(d time.Duration)
}

type realSleeper struct{}

func (realSleeper) Sleep(d time.Duration) { time.Sleep(d) }

// Player replays records with their relative timing.
//
// The provided callback is invoked for each record that contains a frame (Record.Frame != nil).
// START markers are honored by resetting the origin.
//
// speedMultiplier: 1.0 = real time, 2.0 = 2x speed (half waits), 0.5 = half speed.
func Play(records []Record, speedMultiplier float64, loop bool, sleeper Sleeper, cb func(frame []byte) error) error {
	if speedMultiplier <= 0 {
		return fmt.Errorf("speedMultiplier must be > 0")
	}
	if sleeper == nil {
		sleeper = realSleeper{}
	}
	if cb == nil {
		return errors.New("callback is nil")
	}
	if len(records) == 0 {
		return errors.New("no records")
	}

	for {
		var origin time.Duration
		var lastAt time.Duration
		var haveLast bool

		for _, r := range records {
			if r.Frame == nil {
				// START marker.
				origin = r.At
				lastAt = 0
				haveLast = false
				continue
			}

			at := r.At - origin
			if at < 0 {
				at = 0
			}
			if haveLast {
				wait := at - lastAt
				if wait < 0 {
					wait = 0
				}
				wait = time.Duration(float64(wait) / speedMultiplier)
				if wait > 0 {
					sleeper.Sleep(wait)
				}
			}

			if err := cb(r.Frame); err != nil {
				return err
			}

			lastAt = at
			haveLast = true
		}

		if !loop {
			return nil
		}
	}
}
