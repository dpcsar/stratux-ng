package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"stratux-ng/internal/config"
)

func TestRunReplay_SendsFramesInOrder(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "replay.log")
	// Two frames at the same timestamp to avoid sleeps.
	if err := os.WriteFile(path, []byte("0,0102\n0,0a0b0c\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := config.Config{
		GDL90: config.GDL90Config{
			Replay: config.ReplayConfig{
				Enable: true,
				Path:   path,
				Speed:  1.0,
				Loop:   false,
			},
		},
	}

	var sent [][]byte
	err := runReplay(context.Background(), cfg, nil, func(frame []byte) error {
		// Copy to avoid aliasing.
		cp := append([]byte(nil), frame...)
		sent = append(sent, cp)
		return nil
	})
	if err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}

	want := [][]byte{{0x01, 0x02}, {0x0a, 0x0b, 0x0c}}
	if !reflect.DeepEqual(sent, want) {
		t.Fatalf("sent=%x want=%x", sent, want)
	}
}

func TestRunReplay_ContextCanceled_NoSends(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "replay.log")
	if err := os.WriteFile(path, []byte("0,0102\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := config.Config{
		GDL90: config.GDL90Config{
			Replay: config.ReplayConfig{
				Enable: true,
				Path:   path,
				Speed:  1.0,
				Loop:   false,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var sent int
	err := runReplay(ctx, cfg, nil, func(frame []byte) error {
		sent++
		return nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if sent != 0 {
		t.Fatalf("expected 0 sends, got %d", sent)
	}
}
