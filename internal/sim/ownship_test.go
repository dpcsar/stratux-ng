package sim

import (
	"math"
	"testing"
	"time"
)

func TestOwnshipSim_Position_Invariants(t *testing.T) {
	s := OwnshipSim{
		CenterLatDeg: 45.0,
		CenterLonDeg: -122.0,
		RadiusNm:     1.0,
		Period:       60 * time.Second,
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	lat, lon, trk := s.Position(now)

	if math.IsNaN(lat) || math.IsInf(lat, 0) {
		t.Fatalf("lat invalid: %v", lat)
	}
	if math.IsNaN(lon) || math.IsInf(lon, 0) {
		t.Fatalf("lon invalid: %v", lon)
	}
	if math.IsNaN(trk) || math.IsInf(trk, 0) {
		t.Fatalf("track invalid: %v", trk)
	}
	if trk < 0 || trk >= 360 {
		t.Fatalf("track out of range: %v", trk)
	}

	// Rough bound check in degrees (since sim uses small-angle degree math).
	radiusDeg := s.RadiusNm / 60.0
	if math.Abs(lat-s.CenterLatDeg) > radiusDeg*1.01 {
		t.Fatalf("lat offset too large: got %f want <= %f", math.Abs(lat-s.CenterLatDeg), radiusDeg)
	}
	// Lon offset is scaled by cos(lat).
	maxLonDeg := radiusDeg / math.Cos(s.CenterLatDeg*math.Pi/180.0)
	if math.Abs(lon-s.CenterLonDeg) > maxLonDeg*1.01 {
		t.Fatalf("lon offset too large: got %f want <= %f", math.Abs(lon-s.CenterLonDeg), maxLonDeg)
	}
}

func TestOwnshipSim_Position_DeterministicForNow(t *testing.T) {
	s := OwnshipSim{CenterLatDeg: 1, CenterLonDeg: 2, RadiusNm: 0.5, Period: 120 * time.Second}
	now := time.Date(2025, 12, 20, 19, 0, 0, 123, time.UTC)

	lat1, lon1, trk1 := s.Position(now)
	lat2, lon2, trk2 := s.Position(now)
	if lat1 != lat2 || lon1 != lon2 || trk1 != trk2 {
		t.Fatalf("expected deterministic result for same now")
	}
}
