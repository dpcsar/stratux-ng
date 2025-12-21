package sim

import (
	"math"
	"testing"
	"time"
)

func TestTrafficSim_Targets_CountAndInvariants(t *testing.T) {
	s := TrafficSim{
		CenterLatDeg: 45.0,
		CenterLonDeg: -122.0,
		BaseAltFeet:  4500,
		GroundKt:     120,
		RadiusNm:     2.0,
		Period:       90 * time.Second,
	}

	now := time.Date(2025, 12, 20, 19, 0, 0, 0, time.UTC)
	tgts := s.Targets(now, 5)
	if len(tgts) != 5 {
		t.Fatalf("expected 5 targets, got %d", len(tgts))
	}

	radiusDeg := s.RadiusNm / 60.0
	maxLonDeg := radiusDeg / math.Cos(s.CenterLatDeg*math.Pi/180.0)

	for i, tgt := range tgts {
		if math.IsNaN(tgt.LatDeg) || math.IsInf(tgt.LatDeg, 0) {
			t.Fatalf("tgt[%d] lat invalid: %v", i, tgt.LatDeg)
		}
		if math.IsNaN(tgt.LonDeg) || math.IsInf(tgt.LonDeg, 0) {
			t.Fatalf("tgt[%d] lon invalid: %v", i, tgt.LonDeg)
		}
		if tgt.TrackDeg < 0 || tgt.TrackDeg >= 360 {
			t.Fatalf("tgt[%d] track out of range: %v", i, tgt.TrackDeg)
		}
		if tgt.GroundKt < 0 {
			t.Fatalf("tgt[%d] ground speed invalid: %d", i, tgt.GroundKt)
		}

		if math.Abs(tgt.LatDeg-s.CenterLatDeg) > radiusDeg*1.01 {
			t.Fatalf("tgt[%d] lat offset too large", i)
		}
		if math.Abs(tgt.LonDeg-s.CenterLonDeg) > maxLonDeg*1.01 {
			t.Fatalf("tgt[%d] lon offset too large", i)
		}
	}
}

func TestTrafficSim_Targets_ZeroCountNil(t *testing.T) {
	s := TrafficSim{}
	if got := s.Targets(time.Now(), 0); got != nil {
		t.Fatalf("expected nil for count=0")
	}
	if got := s.Targets(time.Now(), -1); got != nil {
		t.Fatalf("expected nil for count<0")
	}
}
