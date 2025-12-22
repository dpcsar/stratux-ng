package gps

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func nmeaLine(payload string) string {
	ck := byte(0)
	for i := 0; i < len(payload); i++ {
		ck ^= payload[i]
	}
	return fmt.Sprintf("$%s*%02X", payload, ck)
}

func TestParseNMEASentence_ChecksumOK(t *testing.T) {
	line := nmeaLine("GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W")
	s, err := parseNMEASentence(line)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if s.Type != "RMC" {
		t.Fatalf("expected type RMC, got %q", s.Type)
	}
}

func TestParseNMEASentence_ChecksumMismatch(t *testing.T) {
	good := nmeaLine("GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W")
	bad := good[:len(good)-2] + "00"
	_, err := parseNMEASentence(bad)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNMEAState_RMCUpdatesFix(t *testing.T) {
	var st nmeaState
	line := nmeaLine("GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W")
	s, err := parseNMEASentence(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := st.apply(now, s)
	if !updated {
		t.Fatalf("expected updated")
	}
	snap := st.snapshot()
	if !snap.Valid {
		t.Fatalf("expected valid")
	}
	if snap.LatDeg == 0 || snap.LonDeg == 0 {
		t.Fatalf("expected non-zero lat/lon")
	}
	if snap.GroundKt == nil {
		t.Fatalf("expected groundspeed")
	}
	if snap.TrackDeg == nil {
		t.Fatalf("expected track")
	}
}

func TestNMEAState_GGAUpdatesAltitude(t *testing.T) {
	var st nmeaState
	line := nmeaLine("GNGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,")
	s, err := parseNMEASentence(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := st.apply(now, s)
	if !updated {
		t.Fatalf("expected updated")
	}
	snap := st.snapshot()
	if snap.AltFeet == nil {
		t.Fatalf("expected altitude")
	}
	if *snap.AltFeet < 1700 || *snap.AltFeet > 1900 {
		t.Fatalf("unexpected alt_feet=%d", *snap.AltFeet)
	}
}

func TestNMEAState_GGAParsesQualitySatsHDOP(t *testing.T) {
	var st nmeaState
	line := nmeaLine("GNGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,")
	s, err := parseNMEASentence(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := st.apply(now, s)
	if !updated {
		t.Fatalf("expected updated")
	}

	snap := st.snapshot()
	if snap.FixQuality == nil || *snap.FixQuality != 1 {
		t.Fatalf("expected fix quality 1, got %+v", snap.FixQuality)
	}
	if snap.Satellites == nil || *snap.Satellites != 8 {
		t.Fatalf("expected satellites 8, got %+v", snap.Satellites)
	}
	if snap.HDOP == nil || math.Abs(*snap.HDOP-0.9) > 1e-6 {
		t.Fatalf("expected hdop 0.9, got %+v", snap.HDOP)
	}
}
