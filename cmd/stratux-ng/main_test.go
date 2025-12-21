package main

import (
	"testing"
	"time"

	"stratux-ng/internal/config"
)

func unframeForMsg(t *testing.T, frame []byte) []byte {
	t.Helper()
	if len(frame) < 4 {
		t.Fatalf("frame too short: %d", len(frame))
	}
	if frame[0] != 0x7E || frame[len(frame)-1] != 0x7E {
		t.Fatalf("missing start/end flags")
	}

	// De-escape and strip flags.
	raw := make([]byte, 0, len(frame))
	for i := 1; i < len(frame)-1; i++ {
		b := frame[i]
		if b == 0x7D {
			i++
			if i >= len(frame)-1 {
				t.Fatalf("truncated escape")
			}
			raw = append(raw, frame[i]^0x20)
			continue
		}
		raw = append(raw, b)
	}
	if len(raw) < 3 {
		t.Fatalf("unescaped payload too short: %d", len(raw))
	}

	// raw = msg + crc16(2 bytes)
	msg := raw[:len(raw)-2]
	if len(msg) == 0 {
		t.Fatalf("empty message")
	}
	return msg
}

func TestBuildGDL90Frames_SimOwnshipAndTrafficMessageSet(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{
			Dest:     "127.0.0.1:4000",
			Interval: 1 * time.Second,
			Mode:     "gdl90",
		},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				Enable:       true,
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
			Traffic: config.TrafficSimConfig{
				Enable:   true,
				Count:    3,
				RadiusNm: 2.0,
				Period:   90 * time.Second,
				GroundKt: 120,
			},
		},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	frames := buildGDL90Frames(cfg, now)
	if len(frames) == 0 {
		t.Fatalf("expected frames")
	}

	counts := map[byte]int{}
	ffSub := map[byte]int{}
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		counts[msg[0]]++
		if msg[0] == 0x65 && len(msg) >= 2 {
			ffSub[msg[1]]++
		}
	}

	// Baseline: heartbeat + stratux hb + ForeFlight messages.
	if counts[0x00] != 1 {
		t.Fatalf("expected 1 heartbeat (0x00), got %d", counts[0x00])
	}
	if counts[0xCC] != 1 {
		t.Fatalf("expected 1 stratux heartbeat (0xCC), got %d", counts[0xCC])
	}
	if ffSub[0x00] != 1 {
		t.Fatalf("expected 1 device ID (0x65/0x00), got %d", ffSub[0x00])
	}
	if ffSub[0x01] != 1 {
		t.Fatalf("expected 1 AHRS message (0x65/0x01), got %d", ffSub[0x01])
	}

	// Ownship + geo-alt.
	if counts[0x0A] != 1 {
		t.Fatalf("expected 1 ownship report (0x0A), got %d", counts[0x0A])
	}
	if counts[0x0B] != 1 {
		t.Fatalf("expected 1 ownship geometric alt (0x0B), got %d", counts[0x0B])
	}

	// Traffic targets.
	if counts[0x14] != cfg.Sim.Traffic.Count {
		t.Fatalf("expected %d traffic reports (0x14), got %d", cfg.Sim.Traffic.Count, counts[0x14])
	}
}
