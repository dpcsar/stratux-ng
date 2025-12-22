package gps

import (
	"math"
	"testing"
	"time"
)

func TestGPSDState_TPVUpdatesFix(t *testing.T) {
	now := time.Date(2025, 12, 22, 12, 0, 0, 0, time.UTC)
	st := newGPSDState("127.0.0.1:2947")

	// speed is m/s when scaled=true; 50 m/s ~= 97.19 kt
	line := `{"class":"TPV","mode":3,"time":"2025-12-22T12:00:00.000Z","lat":45.5,"lon":-122.9,"altMSL":100.0,"speed":50.0,"track":270.0,"climb":1.0,"eph":4.2,"epv":7.0}`
	updated, err := st.applyLine(now, line)
	if err != nil {
		t.Fatalf("applyLine err: %v", err)
	}
	if !updated {
		t.Fatalf("expected updated")
	}

	snap := st.snapshot()
	if !snap.Valid {
		t.Fatalf("expected valid")
	}
	if math.Abs(snap.LatDeg-45.5) > 1e-9 {
		t.Fatalf("lat=%v", snap.LatDeg)
	}
	if math.Abs(snap.LonDeg-(-122.9)) > 1e-9 {
		t.Fatalf("lon=%v", snap.LonDeg)
	}
	if snap.GroundKt == nil {
		t.Fatalf("expected groundspeed")
	}
	if *snap.GroundKt < 96 || *snap.GroundKt > 99 {
		t.Fatalf("ground_kt=%d", *snap.GroundKt)
	}
	if snap.TrackDeg == nil || math.Abs(*snap.TrackDeg-270.0) > 1e-9 {
		t.Fatalf("track=%v", snap.TrackDeg)
	}
	if snap.AltFeet == nil {
		t.Fatalf("expected altitude")
	}
	// 100m ~= 328ft
	if *snap.AltFeet < 320 || *snap.AltFeet > 336 {
		t.Fatalf("alt_feet=%d", *snap.AltFeet)
	}
	if snap.FixMode == nil || *snap.FixMode != 3 {
		t.Fatalf("fix_mode=%v", snap.FixMode)
	}
	if snap.HorizAccM == nil || math.Abs(*snap.HorizAccM-4.2) > 1e-9 {
		t.Fatalf("horiz_acc_m=%v", snap.HorizAccM)
	}
	if snap.VertAccM == nil || math.Abs(*snap.VertAccM-7.0) > 1e-9 {
		t.Fatalf("vert_acc_m=%v", snap.VertAccM)
	}
	if snap.VertSpeedFPM == nil || *snap.VertSpeedFPM < 190 || *snap.VertSpeedFPM > 205 {
		t.Fatalf("vert_speed_fpm=%v", snap.VertSpeedFPM)
	}
	if snap.LastFixUTC == "" {
		t.Fatalf("expected last_fix_utc")
	}
}

func TestGPSDState_SKYUpdatesSatsAndHDOP(t *testing.T) {
	st := newGPSDState("127.0.0.1:2947")
	line := `{"class":"SKY","hdop":0.9,"satellites":[{"used":true},{"used":false},{"used":true}]}`
	updated, err := st.applyLine(time.Now().UTC(), line)
	if err != nil {
		t.Fatalf("applyLine err: %v", err)
	}
	if !updated {
		t.Fatalf("expected updated")
	}
	snap := st.snapshot()
	if snap.Satellites == nil || *snap.Satellites != 2 {
		t.Fatalf("satellites=%v", snap.Satellites)
	}
	if snap.HDOP == nil || math.Abs(*snap.HDOP-0.9) > 1e-9 {
		t.Fatalf("hdop=%v", snap.HDOP)
	}
}
