package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/replay"
)

func TestMsgIDFromFramedGDL90(t *testing.T) {
	frame := gdl90.Frame([]byte{0x0A, 0x01, 0x02, 0x03})
	msgID, ok := msgIDFromFramedGDL90(frame)
	if !ok {
		t.Fatalf("msgIDFromFramedGDL90 ok=false")
	}
	if msgID != 0x0A {
		t.Fatalf("msgID=%02X want %02X", msgID, 0x0A)
	}

	_, ok = msgIDFromFramedGDL90([]byte{0x01, 0x02, 0x03})
	if ok {
		t.Fatalf("expected ok=false for short frame")
	}
}

func TestSummarizeGDL90Log(t *testing.T) {
	f00 := gdl90.Frame([]byte{0x00})
	f0a := gdl90.Frame([]byte{0x0A, 0xFF})
	bad := []byte{0x7E, 0x7D, 0x7E} // malformed escape

	recs := []replay.Record{
		{At: 0, Frame: nil},
		{At: 0 * time.Millisecond, Frame: f00},
		{At: 200 * time.Millisecond, Frame: f0a},
		{At: 300 * time.Millisecond, Frame: bad},
		{At: 0, Frame: nil},
		{At: 1 * time.Second, Frame: f00},
	}

	s := summarizeGDL90Log(recs)
	if s.Segments != 2 {
		t.Fatalf("segments=%d want %d", s.Segments, 2)
	}
	if s.Frames != 4 {
		t.Fatalf("frames=%d want %d", s.Frames, 4)
	}
	if s.Invalid != 1 {
		t.Fatalf("invalid=%d want %d", s.Invalid, 1)
	}
	if s.MsgIDCounts[0x00] != 2 {
		t.Fatalf("count[0x00]=%d want %d", s.MsgIDCounts[0x00], 2)
	}
	if s.MsgIDCounts[0x0A] != 1 {
		t.Fatalf("count[0x0A]=%d want %d", s.MsgIDCounts[0x0A], 1)
	}
	if s.MaxDuration != 1*time.Second {
		t.Fatalf("maxDuration=%s want %s", s.MaxDuration, 1*time.Second)
	}
}

func TestPrintLogSummary_PrintsExpectedFields(t *testing.T) {
	tmp := t.TempDir()
	logPath := tmp + "/gdl90.log"

	w, err := replay.CreateWriter(logPath)
	if err != nil {
		t.Fatalf("CreateWriter() error: %v", err)
	}
	now := time.Now()
	if err := w.WriteFrame(now, gdl90.Frame([]byte{0x00})); err != nil {
		_ = w.Close()
		t.Fatalf("WriteFrame() error: %v", err)
	}
	if err := w.WriteFrame(now, gdl90.Frame([]byte{0x0A, 0x01})); err != nil {
		_ = w.Close()
		t.Fatalf("WriteFrame() error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	oldStdout := os.Stdout
	r, wpipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error: %v", err)
	}
	os.Stdout = wpipe

	printErr := printLogSummary(logPath)

	_ = wpipe.Close()
	os.Stdout = oldStdout

	if printErr != nil {
		_ = r.Close()
		t.Fatalf("printLogSummary() error: %v", printErr)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()
	out := buf.String()

	// Smoke-check for key lines.
	if !bytes.Contains([]byte(out), []byte("path: ")) {
		t.Fatalf("missing path in output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("segments:")) {
		t.Fatalf("missing segments in output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("frames:")) {
		t.Fatalf("missing frames in output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("msg_id_counts:")) {
		t.Fatalf("missing msg_id_counts in output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("0x00: 1")) {
		t.Fatalf("missing 0x00 count in output: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("0x0A: 1")) {
		t.Fatalf("missing 0x0A count in output: %q", out)
	}
}
