package main

import (
	"testing"
	"time"

	"stratux-ng/internal/config"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

func minimalCfg(t *testing.T, dest string, interval time.Duration) config.Config {
	t.Helper()
	cfg := config.Config{
		GDL90: config.GDL90Config{
			Dest:     dest,
			Interval: interval,
		},
	}
	if err := config.DefaultAndValidate(&cfg); err != nil {
		t.Fatalf("DefaultAndValidate() error: %v", err)
	}
	return cfg
}

func TestLiveRuntime_ApplyDestAndInterval(t *testing.T) {
	st := web.NewStatus()

	b, err := udp.NewBroadcaster("127.0.0.1:4000")
	if err != nil {
		t.Fatalf("NewBroadcaster() error: %v", err)
	}
	sender := &safeBroadcaster{b: b}
	defer sender.Close()

	r, err := newLiveRuntime(minimalCfg(t, "127.0.0.1:4000", 1*time.Second), "", st, sender)
	if err != nil {
		t.Fatalf("newLiveRuntime() error: %v", err)
	}
	defer r.Close()

	next := minimalCfg(t, "127.0.0.1:5000", 250*time.Millisecond)
	if err := r.Apply(next); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	got := r.Config()
	if got.GDL90.Dest != "127.0.0.1:5000" {
		t.Fatalf("dest=%q", got.GDL90.Dest)
	}
	if got.GDL90.Interval != 250*time.Millisecond {
		t.Fatalf("interval=%s", got.GDL90.Interval)
	}

	snap := st.Snapshot(time.Now().UTC())
	if snap.GDL90Dest != "127.0.0.1:5000" {
		t.Fatalf("status gdl90_dest=%q", snap.GDL90Dest)
	}
	if snap.Interval != "250ms" {
		t.Fatalf("status interval=%q", snap.Interval)
	}
}

func TestLiveRuntime_RejectsWebListenChange(t *testing.T) {
	st := web.NewStatus()

	b, err := udp.NewBroadcaster("127.0.0.1:4000")
	if err != nil {
		t.Fatalf("NewBroadcaster() error: %v", err)
	}
	sender := &safeBroadcaster{b: b}
	defer sender.Close()

	cfg := minimalCfg(t, "127.0.0.1:4000", 1*time.Second)
	r, err := newLiveRuntime(cfg, "", st, sender)
	if err != nil {
		t.Fatalf("newLiveRuntime() error: %v", err)
	}
	defer r.Close()

	next := cfg
	next.Web.Listen = ":8080"
	if err := r.Apply(next); err == nil {
		t.Fatalf("expected error")
	}
}
