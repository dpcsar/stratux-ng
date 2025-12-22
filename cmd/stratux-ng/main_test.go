package main

import (
	"testing"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gps"
	"stratux-ng/internal/sim"
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
		},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
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
	frames := buildGDL90Frames(cfg, now, false, ahrs.Snapshot{})
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
	ts := sim.TrafficSim{
		CenterLatDeg: cfg.Sim.Ownship.CenterLatDeg,
		CenterLonDeg: cfg.Sim.Ownship.CenterLonDeg,
		BaseAltFeet:  cfg.Sim.Ownship.AltFeet,
		GroundKt:     cfg.Sim.Traffic.GroundKt,
		RadiusNm:     cfg.Sim.Traffic.RadiusNm,
		Period:       cfg.Sim.Traffic.Period,
	}
	visible := 0
	for _, tgt := range ts.Targets(now, cfg.Sim.Traffic.Count) {
		if tgt.Visible {
			visible++
		}
	}
	if counts[0x14] != visible {
		t.Fatalf("expected %d visible traffic reports (0x14), got %d", visible, counts[0x14])
	}
}

func TestBuildGDL90Frames_OwnshipAltitudePrefersBaroPressureAltitude(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{
			Dest:     "127.0.0.1:4000",
			Interval: 1 * time.Second,
		},
		AHRS: config.AHRSConfig{Enable: true},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
		},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)

	// Provide a baro pressure altitude that differs from the sim GPS altitude.
	// Use a 25-ft-aligned value to avoid GDL90 quantization ambiguity.
	baroAltFeet := 4150.0
	frames := buildGDL90Frames(cfg, now, true, ahrs.Snapshot{
		Valid:            true,
		IMUDetected:      true,
		BaroDetected:     true,
		PressureAltFeet:  baroAltFeet,
		PressureAltValid: true,
	})

	var ownshipMsg []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if msg[0] == 0x0A {
			ownshipMsg = msg
			break
		}
	}
	if ownshipMsg == nil {
		t.Fatalf("expected ownship report (0x0A)")
	}
	if len(ownshipMsg) < 13 {
		t.Fatalf("ownship msg too short: %d", len(ownshipMsg))
	}

	// Decode the 12-bit altitude field (25 ft resolution with +1000 ft offset).
	enc := (uint16(ownshipMsg[11]) << 4) | (uint16(ownshipMsg[12]) >> 4)
	if enc == 0x0FFF {
		t.Fatalf("expected valid altitude encoding")
	}
	decodedAltFeet := int(enc)*25 - 1000

	if decodedAltFeet != int(baroAltFeet) {
		t.Fatalf("expected ownship altitude=%d (baro), got %d", int(baroAltFeet), decodedAltFeet)
	}
}

func TestBuildGDL90Frames_DoesNotAdvertiseGPSValidWithoutOwnship(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{
			Dest:     "127.0.0.1:4000",
			Interval: 1 * time.Second,
		},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "BADICAO",
				Callsign:     "STRATUX",
			},
		},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	frames := buildGDL90Frames(cfg, now, false, ahrs.Snapshot{})
	if len(frames) == 0 {
		t.Fatalf("expected frames")
	}

	var hb00 []byte
	var hbCC []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		switch msg[0] {
		case 0x00:
			hb00 = msg
		case 0xCC:
			hbCC = msg
		case 0x0A:
			t.Fatalf("did not expect ownship report (0x0A) when ICAO is invalid")
		}
	}
	if hb00 == nil {
		t.Fatalf("expected heartbeat (0x00)")
	}
	if len(hb00) < 2 {
		t.Fatalf("heartbeat too short: %d", len(hb00))
	}
	// In our heartbeat implementation, msg[1] bit7 indicates UTC/GPS OK.
	if (hb00[1] & 0x80) != 0 {
		t.Fatalf("expected heartbeat gpsValid=false")
	}

	if hbCC == nil {
		t.Fatalf("expected stratux heartbeat (0xCC)")
	}
	if len(hbCC) < 2 {
		t.Fatalf("stratux heartbeat too short: %d", len(hbCC))
	}
	// In stratux heartbeat, msg[1] bit1 indicates GPS valid.
	if (hbCC[1] & 0x02) != 0 {
		t.Fatalf("expected stratux heartbeat gpsValid=false")
	}
}

func TestBuildGDL90FramesWithGPS_OwnshipVerticalSpeedFromGPS(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
			Traffic: config.TrafficSimConfig{Enable: false},
		},
	}

	now := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 270.0
	vv := 640 // 10 * 64 fpm
	gpsSnap := gps.Snapshot{
		Enabled:      true,
		Valid:        true,
		LatDeg:       45.5,
		LonDeg:       -122.9,
		AltFeet:      &alt,
		GroundKt:     &gs,
		TrackDeg:     &trk,
		VertSpeedFPM: &vv,
		LastFixUTC:   now.UTC().Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap)
	var ownshipMsg []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if msg[0] == 0x0A {
			ownshipMsg = msg
			break
		}
	}
	if ownshipMsg == nil {
		t.Fatalf("expected ownship report")
	}
	if len(ownshipMsg) < 17 {
		t.Fatalf("ownship msg too short: %d", len(ownshipMsg))
	}

	vvelU := uint16(ownshipMsg[15]&0x0F)<<8 | uint16(ownshipMsg[16])
	if vvelU != 0x00A {
		t.Fatalf("unexpected vvel: got 0x%03X want 0x00A", vvelU)
	}
}

func TestBuildGDL90FramesWithGPS_OwnshipVerticalSpeedNegativeFromGPS(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
			Traffic: config.TrafficSimConfig{Enable: false},
		},
	}

	now := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 270.0
	vv := -640 // -10 * 64 fpm
	gpsSnap := gps.Snapshot{
		Enabled:      true,
		Valid:        true,
		LatDeg:       45.5,
		LonDeg:       -122.9,
		AltFeet:      &alt,
		GroundKt:     &gs,
		TrackDeg:     &trk,
		VertSpeedFPM: &vv,
		LastFixUTC:   now.UTC().Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap)
	var ownshipMsg []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if msg[0] == 0x0A {
			ownshipMsg = msg
			break
		}
	}
	if ownshipMsg == nil {
		t.Fatalf("expected ownship report")
	}
	if len(ownshipMsg) < 17 {
		t.Fatalf("ownship msg too short: %d", len(ownshipMsg))
	}

	vvelU := uint16(ownshipMsg[15]&0x0F)<<8 | uint16(ownshipMsg[16])
	if vvelU != 0xFF6 {
		t.Fatalf("unexpected vvel: got 0x%03X want 0xFF6", vvelU)
	}
}

func TestBuildGDL90FramesWithGPS_OwnshipVerticalSpeedUnknownWhenAbsent(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
			Traffic: config.TrafficSimConfig{Enable: false},
		},
	}

	now := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 270.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.5,
		LonDeg:     -122.9,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		LastFixUTC: now.UTC().Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap)
	var ownshipMsg []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if msg[0] == 0x0A {
			ownshipMsg = msg
			break
		}
	}
	if ownshipMsg == nil {
		t.Fatalf("expected ownship report")
	}
	if len(ownshipMsg) < 17 {
		t.Fatalf("ownship msg too short: %d", len(ownshipMsg))
	}

	vvelU := uint16(ownshipMsg[15]&0x0F)<<8 | uint16(ownshipMsg[16])
	if vvelU != 0x800 {
		t.Fatalf("unexpected vvel: got 0x%03X want 0x800 (unknown)", vvelU)
	}
}

func TestBuildGDL90FramesWithGPS_AHRSLEVerticalSpeedFromGPS(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: false},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				CenterLatDeg: 45.541,
				CenterLonDeg: -122.949,
				AltFeet:      3500,
				GroundKt:     90,
				RadiusNm:     0.5,
				Period:       120 * time.Second,
				ICAO:         "F00001",
				Callsign:     "STRATUX",
			},
			Traffic: config.TrafficSimConfig{Enable: false},
		},
	}

	now := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 270.0
	vv := 640
	gpsSnap := gps.Snapshot{
		Enabled:      true,
		Valid:        true,
		LatDeg:       45.5,
		LonDeg:       -122.9,
		AltFeet:      &alt,
		GroundKt:     &gs,
		TrackDeg:     &trk,
		VertSpeedFPM: &vv,
		LastFixUTC:   now.UTC().Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap)
	var leMsg []byte
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if len(msg) >= 22 && msg[0] == 0x4C {
			leMsg = msg
			break
		}
	}
	if leMsg == nil {
		t.Fatalf("expected AHRS LE report (0x4C)")
	}

	vs := int16(leMsg[20])<<8 | int16(leMsg[21])
	if vs != 640 {
		t.Fatalf("unexpected LE vertical speed: got %d want 640", vs)
	}
}
