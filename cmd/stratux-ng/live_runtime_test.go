package main

import (
	"context"
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

	r, err := newLiveRuntime(context.Background(), minimalCfg(t, "127.0.0.1:4000", 1*time.Second), "", st, sender)
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
	r, err := newLiveRuntime(context.Background(), cfg, "", st, sender)
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

// TestLiveRuntime_ApplyUnrelatedChangeDoesNotRequireRestartAfterSDRAutodetect
// guards against a regression where SDR auto-detection mutates the running
// ADSB1090/UAT978 config (resolving "auto" and upserting --device args) and
// a subsequent Apply() of an unrelated change (e.g. GDL90 interval) falsely
// reported "adsb1090/uat978 settings require restart" because it compared
// against the already-mutated runtime config instead of the pre-mutation one.
func TestLiveRuntime_ApplyUnrelatedChangeDoesNotRequireRestartAfterSDRAutodetect(t *testing.T) {
	st := web.NewStatus()

	b, err := udp.NewBroadcaster("127.0.0.1:4000")
	if err != nil {
		t.Fatalf("NewBroadcaster() error: %v", err)
	}
	sender := &safeBroadcaster{b: b}
	defer sender.Close()

	cfg := minimalCfg(t, "127.0.0.1:4000", 1*time.Second)
	cfg.ADSB1090 = config.DecoderBandConfig{
		Enable: true,
		Decoder: config.DecoderConfig{
			Command:    "dump1090-fa",
			Args:       []string{"--net-stratux-port", "30006"},
			JSONListen: "127.0.0.1:0",
		},
		SDR: config.SDRSelector{SerialTag: "auto"},
	}

	r, err := newLiveRuntime(context.Background(), cfg, "", st, sender)
	if err != nil {
		t.Fatalf("newLiveRuntime() error: %v", err)
	}
	defer r.Close()

	// Simulate a freshly-submitted config from a settings/aircraft save: same
	// logical adsb1090 config as originally loaded, but with an unrelated
	// change (GDL90 interval).
	next := r.Config()
	next.ADSB1090 = cfg.ADSB1090
	next.GDL90.Interval = 250 * time.Millisecond

	if err := r.Apply(next); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
}
