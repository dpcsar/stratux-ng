package replay

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"stratux-ng/internal/gdl90"
)

func TestRecordReplay_RoundTripFramesInOrder(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "gdl90-record.log")

	w, err := CreateWriter(path)
	if err != nil {
		t.Fatalf("CreateWriter() error: %v", err)
	}

	// Use the same timestamp for every frame so replay has zero waits.
	now := time.Now()

	framesIn := [][]byte{
		gdl90.HeartbeatFrame(true, false),
		gdl90.StratuxHeartbeatFrame(true, false),
		gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"),
	}
	for _, f := range framesIn {
		if err := w.WriteFrame(now, f); err != nil {
			_ = w.Close()
			t.Fatalf("WriteFrame() error: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	rc, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer rc.Close()

	recs, err := NewReader(rc).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}

	var framesOut [][]byte
	fs := &fakeSleeper{}
	err = Play(recs, 1.0, false, fs, func(frame []byte) error {
		cp := append([]byte(nil), frame...)
		framesOut = append(framesOut, cp)
		return nil
	})
	if err != nil {
		t.Fatalf("Play() error: %v", err)
	}

	if len(fs.slept) != 0 {
		t.Fatalf("expected no sleeps, got %v", fs.slept)
	}

	if !reflect.DeepEqual(framesOut, framesIn) {
		t.Fatalf("frames mismatch\n got: %x\nwant: %x", framesOut, framesIn)
	}
}
