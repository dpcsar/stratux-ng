package main

import (
	"testing"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, nil)
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, nil)
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, nil)
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, nil)
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

func findAHRSLERaw(t *testing.T, frames [][]byte) []byte {
	t.Helper()
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if len(msg) >= 10 && msg[0] == 0x4C {
			return msg
		}
	}
	return nil
}

func leHeadingDeg(t *testing.T, leMsg []byte) float64 {
	t.Helper()
	if len(leMsg) < 10 {
		t.Fatalf("LE msg too short")
	}
	h10 := int16(leMsg[8])<<8 | int16(leMsg[9])
	if h10 == 0x7FFF {
		t.Fatalf("LE heading invalid")
	}
	return float64(h10) / 10.0
}

func TestBuildGDL90FramesWithGPS_HeadingUsesYawForShortTurnsAndGPSForAccuracy(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 200 * time.Millisecond},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
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

	t0 := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 90.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.5,
		LonDeg:     -122.9,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		LastFixUTC: t0.UTC().Format(time.RFC3339Nano),
	}

	hf := &headingFuser{}
	if _, err := gdl90.ParseICAOHex(cfg.Sim.Ownship.ICAO); err != nil {
		t.Fatalf("test config ICAO=%q invalid: %v", cfg.Sim.Ownship.ICAO, err)
	}
	if gpsSnap.LastFixUTC == "" {
		t.Fatalf("test gpsSnap.LastFixUTC empty")
	}
	if tFix, err := time.Parse(time.RFC3339Nano, gpsSnap.LastFixUTC); err != nil {
		t.Fatalf("test gpsSnap.LastFixUTC=%q parse failed: %v", gpsSnap.LastFixUTC, err)
	} else if t0.UTC().Sub(tFix.UTC()) > 3*time.Second {
		t.Fatalf("test gps fix stale: now=%s fix=%s", t0.UTC().Format(time.RFC3339Nano), tFix.UTC().Format(time.RFC3339Nano))
	}

	// Seed: heading should start at GPS track.
	frames0 := buildGDL90FramesWithGPS(cfg, t0, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
	le0 := findAHRSLERaw(t, frames0)
	h0 := leHeadingDeg(t, le0)
	if h0 < 89.9 || h0 > 90.1 {
		t.Fatalf("seed heading=%v want ~90", h0)
	}

	// Short turn: yaw-rate should advance heading quickly even if GPS track is unchanged.
	t1 := t0.Add(200 * time.Millisecond)
	gpsSnap.LastFixUTC = t1.UTC().Format(time.RFC3339Nano)
	frames1 := buildGDL90FramesWithGPS(cfg, t1, true, ahrs.Snapshot{Valid: true, YawRateDps: 30}, true, gpsSnap, hf, nil)
	le1 := findAHRSLERaw(t, frames1)
	h1 := leHeadingDeg(t, le1)
	if h1 <= 92 {
		t.Fatalf("turn heading=%v want >92", h1)
	}

	// Accuracy: with no yaw-rate, heading should converge back toward GPS track over several ticks.
	now := t1
	for i := 0; i < 10; i++ {
		now = now.Add(500 * time.Millisecond)
		gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
		frames := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
		_ = findAHRSLERaw(t, frames)
	}
	now = now.Add(500 * time.Millisecond)
	gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
	framesF := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
	leF := findAHRSLERaw(t, framesF)
	hF := leHeadingDeg(t, leF)
	if hF < 88 || hF > 92 {
		t.Fatalf("final heading=%v want near 90", hF)
	}
}

func TestBuildGDL90FramesWithGPS_DoesNotCorrectWhenFixModeInvalid(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 200 * time.Millisecond},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
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

	t0 := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	alt := 5000
	gs := 100
	trk := 90.0
	fixMode := 1 // <2 => treat track as invalid for correction
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.5,
		LonDeg:     -122.9,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		FixMode:    &fixMode,
		LastFixUTC: t0.UTC().Format(time.RFC3339Nano),
	}

	hf := &headingFuser{}
	frames0 := buildGDL90FramesWithGPS(cfg, t0, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
	h0 := leHeadingDeg(t, findAHRSLERaw(t, frames0))
	if h0 < 89.9 || h0 > 90.1 {
		t.Fatalf("seed heading=%v want ~90", h0)
	}

	// Turn: yaw moves heading away from GPS track.
	t1 := t0.Add(200 * time.Millisecond)
	gpsSnap.LastFixUTC = t1.UTC().Format(time.RFC3339Nano)
	frames1 := buildGDL90FramesWithGPS(cfg, t1, true, ahrs.Snapshot{Valid: true, YawRateDps: 30}, true, gpsSnap, hf, nil)
	h1 := leHeadingDeg(t, findAHRSLERaw(t, frames1))
	if h1 <= 92 {
		t.Fatalf("turn heading=%v want >92", h1)
	}

	// With GPS track invalid-for-correction, heading should not converge back toward 90.
	now := t1
	for i := 0; i < 12; i++ {
		now = now.Add(500 * time.Millisecond)
		gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
		frames := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
		_ = findAHRSLERaw(t, frames)
	}
	framesF := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf, nil)
	hF := leHeadingDeg(t, findAHRSLERaw(t, framesF))
	if hF < 92 {
		t.Fatalf("final heading=%v unexpectedly converged toward GPS track", hF)
	}
}

func TestHeadingFuser_WrapAround(t *testing.T) {
	hf := &headingFuser{}
	t0 := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	trk := 350.0
	gs := 100
	// Seed.
	_ = hf.Update(t0, &trk, true, gs, nil)
	// Turn right 60 dps for 0.5s => +30 deg => wraps to ~20.
	t1 := t0.Add(500 * time.Millisecond)
	yaw := 60.0
	h := hf.Update(t1, &trk, true, gs, &yaw)
	if h < 5 || h > 40 {
		t.Fatalf("wrapped heading=%v want around 20", h)
	}
}

func TestHeadingFuser_DoesNotCorrectWhenGPSTrackInvalid(t *testing.T) {
	hf := &headingFuser{}
	t0 := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	trk := 90.0
	gs := 100
	_ = hf.Update(t0, &trk, true, gs, nil)

	// Turn: integrate yaw to move away from GPS track.
	t1 := t0.Add(200 * time.Millisecond)
	yaw := 30.0
	h1 := hf.Update(t1, &trk, false, gs, &yaw) // gpsTrackValid=false => no correction
	if h1 <= 92 {
		t.Fatalf("heading after turn=%v want >92", h1)
	}

	// With no yaw and GPS still invalid, it should *not* converge back quickly.
	t2 := t1.Add(2 * time.Second)
	h2 := hf.Update(t2, &trk, false, gs, nil)
	if h2 < h1-0.5 {
		t.Fatalf("heading corrected unexpectedly: before=%v after=%v", h1, h2)
	}
}

func TestBuildGDL90FramesWithGPS_AppendsLiveTrafficWhenSimTrafficDisabled(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Sim: config.SimConfig{
			Ownship: config.OwnshipSimConfig{
				ICAO:     "F00001",
				Callsign: "STRATUX",
				AltFeet:   3500,
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

	icaoT, err := gdl90.ParseICAOHex("ABC123")
	if err != nil {
		t.Fatalf("test ICAO invalid: %v", err)
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, []gdl90.Traffic{
		{AddrType: 0x00, ICAO: icaoT, LatDeg: 45.6, LonDeg: -122.8, AltFeet: 4200, NIC: 8, NACp: 8, GroundKt: 120, TrackDeg: 180, VvelFpm: 0, OnGround: false, EmitterCategory: 0x01, Tail: "N12345"},
	})

	var found bool
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		if len(msg) >= 5 && msg[0] == 0x14 && msg[2] == 0xAB && msg[3] == 0xC1 && msg[4] == 0x23 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected traffic report for ABC123")
	}
}
