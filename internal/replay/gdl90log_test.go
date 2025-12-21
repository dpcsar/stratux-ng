package replay

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeSleeper struct {
	slept []time.Duration
}

func (fs *fakeSleeper) Sleep(d time.Duration) {
	fs.slept = append(fs.slept, d)
}

func TestReaderReadAll(t *testing.T) {
	in := strings.NewReader(`
# comment

START
0, 0102
10, 0a 0b
`)

	recs, err := NewReader(in).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(recs))
	}
	if recs[0].Frame != nil {
		t.Fatalf("expected START marker (nil frame), got %v", recs[0].Frame)
	}
	if recs[1].At != 0 {
		t.Fatalf("expected At=0, got %s", recs[1].At)
	}
	if !reflect.DeepEqual(recs[1].Frame, []byte{0x01, 0x02}) {
		t.Fatalf("unexpected frame 1: %x", recs[1].Frame)
	}
	if recs[2].At != 10*time.Nanosecond {
		t.Fatalf("expected At=10ns, got %s", recs[2].At)
	}
	if !reflect.DeepEqual(recs[2].Frame, []byte{0x0a, 0x0b}) {
		t.Fatalf("unexpected frame 2: %x", recs[2].Frame)
	}
}

func TestReaderReadAll_InvalidLine(t *testing.T) {
	in := strings.NewReader("not-a-valid-line\n")
	_, err := NewReader(in).ReadAll()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlay_RespectsTimingAndStart(t *testing.T) {
	frames := make([][]byte, 0, 3)
	fs := &fakeSleeper{}

	recs := []Record{
		{At: 1 * time.Second, Frame: nil},
		{At: 1 * time.Second, Frame: []byte{0xAA}},
		{At: 1*time.Second + 100*time.Nanosecond, Frame: []byte{0xBB}},
		{At: 2 * time.Second, Frame: nil},
		{At: 2*time.Second + 50*time.Nanosecond, Frame: []byte{0xCC}},
	}

	err := Play(recs, 1.0, false, fs, func(frame []byte) error {
		cp := append([]byte(nil), frame...)
		frames = append(frames, cp)
		return nil
	})
	if err != nil {
		t.Fatalf("Play() error: %v", err)
	}

	wantFrames := [][]byte{{0xAA}, {0xBB}, {0xCC}}
	if len(frames) != len(wantFrames) {
		t.Fatalf("expected %d frames, got %d", len(wantFrames), len(frames))
	}
	for i := range wantFrames {
		if !reflect.DeepEqual(frames[i], wantFrames[i]) {
			t.Fatalf("frame[%d] = %x, want %x", i, frames[i], wantFrames[i])
		}
	}

	if !reflect.DeepEqual(fs.slept, []time.Duration{100 * time.Nanosecond}) {
		t.Fatalf("slept = %v, want [100ns]", fs.slept)
	}
}

func TestPlay_SpeedMultiplier(t *testing.T) {
	fs := &fakeSleeper{}
	recs := []Record{
		{At: 0, Frame: []byte{0x01}},
		{At: 100 * time.Nanosecond, Frame: []byte{0x02}},
	}

	err := Play(recs, 2.0, false, fs, func(frame []byte) error { return nil })
	if err != nil {
		t.Fatalf("Play() error: %v", err)
	}
	if !reflect.DeepEqual(fs.slept, []time.Duration{50 * time.Nanosecond}) {
		t.Fatalf("slept = %v, want [50ns]", fs.slept)
	}
}

func TestPlay_InvalidSpeed(t *testing.T) {
	recs := []Record{{At: 0, Frame: []byte{0x01}}}
	if err := Play(recs, 0, false, nil, func([]byte) error { return nil }); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriter_WritesExpectedFormat(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.log")

	w, err := CreateWriter(path)
	if err != nil {
		t.Fatalf("CreateWriter() error: %v", err)
	}
	w.start = time.Unix(0, 0)

	if err := w.WriteFrame(time.Unix(0, 20), []byte{0x01, 0x02}); err != nil {
		t.Fatalf("WriteFrame() error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(b) != "START\n20,0102\n" {
		t.Fatalf("unexpected file contents: %q", string(b))
	}
}
