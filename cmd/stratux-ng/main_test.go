package main

import (
	"testing"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/gps"
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

func mustParseICAO(t *testing.T, hex string) [3]byte {
	t.Helper()
	icao, err := gdl90.ParseICAOHex(hex)
	if err != nil {
		t.Fatalf("ParseICAOHex %q: %v", hex, err)
	}
	return icao
}

func TestBuildGDL90FramesWithGPS_MessageSet(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
		},
		AHRS: config.AHRSConfig{Enable: true},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	alt := 4800
	ground := 140
	track := 45.0
	vv := 0
	gpsSnap := gps.Snapshot{
		Enabled:      true,
		Valid:        true,
		LatDeg:       45.5,
		LonDeg:       -122.9,
		AltFeet:      &alt,
		GroundKt:     &ground,
		TrackDeg:     &track,
		VertSpeedFPM: &vv,
		LastFixUTC:   now.UTC().Format(time.RFC3339Nano),
	}

	liveTraffic := []gdl90.Traffic{
		{AddrType: 0x00, ICAO: mustParseICAO(t, "ABC001"), LatDeg: 45.6, LonDeg: -122.8, AltFeet: 5200, NIC: 8, NACp: 8, GroundKt: 150, TrackDeg: 90, VvelFpm: 0, OnGround: false, Tail: "N00001", EmitterCategory: 0x01},
		{AddrType: 0x00, ICAO: mustParseICAO(t, "ABC002"), LatDeg: 45.4, LonDeg: -122.7, AltFeet: 4100, NIC: 8, NACp: 7, GroundKt: 120, TrackDeg: 180, VvelFpm: -256, OnGround: false, Tail: "N00002", EmitterCategory: 0x02},
	}

	frames := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{
		Valid:            true,
		PressureAltValid: true,
		PressureAltFeet:  4700,
		RollDeg:          1.0,
		PitchDeg:         -0.5,
	}, true, gpsSnap, liveTraffic)
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

	if counts[0x00] != 1 {
		t.Fatalf("expected 1 heartbeat (0x00), got %d", counts[0x00])
	}
	if counts[0xCC] != 1 {
		t.Fatalf("expected 1 stratux heartbeat (0xCC), got %d", counts[0xCC])
	}
	if ffSub[0x00] != 1 {
		t.Fatalf("expected 1 device ID (0x65/0x00), got %d", ffSub[0x00])
	}
	if counts[0x0A] != 1 {
		t.Fatalf("expected 1 ownship report (0x0A), got %d", counts[0x0A])
	}
	if counts[0x0B] != 1 {
		t.Fatalf("expected 1 ownship geometric alt (0x0B), got %d", counts[0x0B])
	}
	if counts[0x14] != len(liveTraffic) {
		t.Fatalf("expected %d traffic reports (0x14), got %d", len(liveTraffic), counts[0x14])
	}
}

func TestBuildGDL90Frames_OwnshipAltitudePrefersBaroPressureAltitude(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
		},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	alt := 3500
	ground := 90
	track := 90.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.5,
		LonDeg:     -122.9,
		AltFeet:    &alt,
		GroundKt:   &ground,
		TrackDeg:   &track,
		LastFixUTC: now.UTC().Format(time.RFC3339Nano),
	}

	// Provide a baro pressure altitude that differs from the sim GPS altitude.
	// Use a 25-ft-aligned value to avoid GDL90 quantization ambiguity.
	baroAltFeet := 4150.0
	frames := buildGDL90FramesWithGPS(cfg, now, true, ahrs.Snapshot{
		Valid:            true,
		IMUDetected:      true,
		BaroDetected:     true,
		PressureAltFeet:  baroAltFeet,
		PressureAltValid: true,
	}, true, gpsSnap, nil)

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
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Ownship: config.OwnshipConfig{
			ICAO:     "BADICAO",
			Callsign: "STRATUX",
		},
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	alt := 4000
	ground := 100
	track := 270.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.5,
		LonDeg:     -122.9,
		AltFeet:    &alt,
		GroundKt:   &ground,
		TrackDeg:   &track,
		LastFixUTC: now.UTC().Format(time.RFC3339Nano),
	}
	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, nil)
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
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, nil)
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
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, nil)
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
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, nil)
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

func TestAttitudeLEFrame_VerticalSpeedFromGPS(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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

	att := buildAttitudePayload(cfg, now, true, ahrs.Snapshot{Valid: true}, true, gpsSnap, &headingFuser{})
	if att.VerticalSpeedFpm != vv || !att.VerticalSpeedValid {
		t.Fatalf("attitude vertical speed invalid: att=%+v", att)
	}
	frame := gdl90.AHRSGDL90LEFrame(att)
	leMsg := unframeForMsg(t, frame)
	if len(leMsg) < 22 || leMsg[0] != 0x4C {
		t.Fatalf("unexpected LE frame: %v", leMsg)
	}
	vs := int16(leMsg[20])<<8 | int16(leMsg[21])
	if vs != int16(vv) {
		t.Fatalf("unexpected LE vertical speed: got %d want %d", vs, vv)
	}
}

func TestBuildAttitudePayload_HeadingUsesYawForShortTurnsAndGPSForAccuracy(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 200 * time.Millisecond},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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
	att0 := buildAttitudePayload(cfg, t0, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	if att0.HeadingDeg < 89.9 || att0.HeadingDeg > 90.1 {
		t.Fatalf("seed heading=%v want ~90", att0.HeadingDeg)
	}

	// Short turn: yaw-rate should advance heading quickly even if GPS track is unchanged.
	t1 := t0.Add(200 * time.Millisecond)
	gpsSnap.LastFixUTC = t1.UTC().Format(time.RFC3339Nano)
	att1 := buildAttitudePayload(cfg, t1, true, ahrs.Snapshot{Valid: true, YawRateDps: 30}, true, gpsSnap, hf)
	if att1.HeadingDeg <= 92 {
		t.Fatalf("turn heading=%v want >92", att1.HeadingDeg)
	}

	// Accuracy: with no yaw-rate, heading should converge back toward GPS track over several ticks.
	now := t1
	for i := 0; i < 10; i++ {
		now = now.Add(500 * time.Millisecond)
		gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
		_ = buildAttitudePayload(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	}
	now = now.Add(500 * time.Millisecond)
	gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
	attFinal := buildAttitudePayload(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	if attFinal.HeadingDeg < 88 || attFinal.HeadingDeg > 92 {
		t.Fatalf("final heading=%v want near 90", attFinal.HeadingDeg)
	}
}

func TestBuildAttitudePayload_DoesNotCorrectWhenFixModeInvalid(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 200 * time.Millisecond},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		AHRS:  config.AHRSConfig{Enable: true},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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
	att0 := buildAttitudePayload(cfg, t0, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	if att0.HeadingDeg < 89.9 || att0.HeadingDeg > 90.1 {
		t.Fatalf("seed heading=%v want ~90", att0.HeadingDeg)
	}

	// Turn: yaw moves heading away from GPS track.
	t1 := t0.Add(200 * time.Millisecond)
	gpsSnap.LastFixUTC = t1.UTC().Format(time.RFC3339Nano)
	att1 := buildAttitudePayload(cfg, t1, true, ahrs.Snapshot{Valid: true, YawRateDps: 30}, true, gpsSnap, hf)
	if att1.HeadingDeg <= 92 {
		t.Fatalf("turn heading=%v want >92", att1.HeadingDeg)
	}

	// With GPS track invalid-for-correction, heading should not converge back toward 90.
	now := t1
	for i := 0; i < 12; i++ {
		now = now.Add(500 * time.Millisecond)
		gpsSnap.LastFixUTC = now.UTC().Format(time.RFC3339Nano)
		_ = buildAttitudePayload(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	}
	attFinal := buildAttitudePayload(cfg, now, true, ahrs.Snapshot{Valid: true, YawRateDps: 0}, true, gpsSnap, hf)
	if attFinal.HeadingDeg < 92 {
		t.Fatalf("final heading=%v unexpectedly converged toward GPS track", attFinal.HeadingDeg)
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

func TestBuildGDL90FramesWithGPS_AppendsLiveTraffic(t *testing.T) {
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Ownship: config.OwnshipConfig{
			ICAO:     "F00001",
			Callsign: "STRATUX",
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

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, []gdl90.Traffic{
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

func TestAttitudeSnapshotFromPayload_PrefersAHRSButKeepsHeading(t *testing.T) {
	snap := ahrs.Snapshot{
		Valid:            true,
		IMUDetected:      true,
		StartupReady:     true,
		PressureAltValid: true,
		PressureAltFeet:  3150,
		RollDeg:          1.5,
		PitchDeg:         -2.0,
		GLoadValid:       true,
		GLoadG:           1.15,
	}
	att := gdl90.Attitude{
		Valid:                true,
		RollDeg:              10,
		PitchDeg:             -5,
		HeadingDeg:           123.4,
		GLoad:                0.98,
		PressureAltitudeFeet: 3421,
		PressureAltValid:     true,
	}
	out := attitudeSnapshotFromPayload(att, true, snap)
	if out.RollDeg == nil || *out.RollDeg != snap.RollDeg {
		t.Fatalf("expected roll from AHRS, got %+v", out.RollDeg)
	}
	if out.PitchDeg == nil || *out.PitchDeg != snap.PitchDeg {
		t.Fatalf("expected pitch from AHRS, got %+v", out.PitchDeg)
	}
	if out.GLoad == nil || *out.GLoad != snap.GLoadG {
		t.Fatalf("expected g-load from AHRS, got %+v", out.GLoad)
	}
	if out.PressureAltFt == nil || *out.PressureAltFt != snap.PressureAltFeet {
		t.Fatalf("expected pressure altitude from AHRS, got %+v", out.PressureAltFt)
	}
	if out.HeadingDeg == nil || *out.HeadingDeg != att.HeadingDeg {
		t.Fatalf("expected heading from payload, got %+v", out.HeadingDeg)
	}
}

func TestAttitudeSnapshotFromPayload_FallsBackWhenAHRSInvalid(t *testing.T) {
	snap := ahrs.Snapshot{
		Valid:        false,
		IMUDetected:  true,
		StartupReady: false,
	}
	att := gdl90.Attitude{
		Valid:                true,
		RollDeg:              -7.5,
		PitchDeg:             4.25,
		HeadingDeg:           212.0,
		GLoad:                1.22,
		PressureAltitudeFeet: 2780,
		PressureAltValid:     true,
	}
	out := attitudeSnapshotFromPayload(att, true, snap)
	if out.Valid {
		t.Fatalf("expected snapshot to remain invalid when AHRS is unavailable")
	}
	if out.RollDeg == nil || *out.RollDeg != att.RollDeg {
		t.Fatalf("expected roll fallback from payload, got %+v", out.RollDeg)
	}
	if out.PitchDeg == nil || *out.PitchDeg != att.PitchDeg {
		t.Fatalf("expected pitch fallback from payload, got %+v", out.PitchDeg)
	}
	if out.GLoad == nil || *out.GLoad != att.GLoad {
		t.Fatalf("expected g-load fallback from payload, got %+v", out.GLoad)
	}
	if out.PressureAltFt == nil || *out.PressureAltFt != att.PressureAltitudeFeet {
		t.Fatalf("expected pressure altitude fallback from payload, got %+v", out.PressureAltFt)
	}
	if out.HeadingDeg == nil || *out.HeadingDeg != att.HeadingDeg {
		t.Fatalf("expected heading from payload, got %+v", out.HeadingDeg)
	}
}
