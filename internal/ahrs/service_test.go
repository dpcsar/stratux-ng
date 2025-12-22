package ahrs

import (
	"math"
	"testing"
)

func TestDominantAxis(t *testing.T) {
	if got := dominantAxis(0.9, 0.1, 0.2); got != 1 {
		t.Fatalf("got=%d want=1", got)
	}
	if got := dominantAxis(-0.9, 0.1, 0.2); got != -1 {
		t.Fatalf("got=%d want=-1", got)
	}
	if got := dominantAxis(0.1, -0.8, 0.2); got != -2 {
		t.Fatalf("got=%d want=-2", got)
	}
	if got := dominantAxis(0.1, 0.2, 0.3); got != 3 {
		t.Fatalf("got=%d want=3", got)
	}
	// Tie-break: prefers X when equal.
	if got := dominantAxis(1, 1, 0); got != 1 {
		t.Fatalf("got=%d want=1", got)
	}
}

func TestPressureToAltitudeFeet_SeaLevel(t *testing.T) {
	alt := pressureToAltitudeFeet(101325.0)
	if math.Abs(alt) > 1.0 {
		t.Fatalf("alt=%v want ~0", alt)
	}
}

func TestApplyOrientationFromGravity_Identity(t *testing.T) {
	s := New(Config{Enable: false})
	s.forwardAxis = 1
	if err := s.applyOrientationFromGravity([3]float64{0, 0, 1}); err != nil {
		t.Fatalf("applyOrientationFromGravity: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.orientationSet {
		t.Fatalf("expected orientationSet")
	}
	// Expect body Z aligns with gravity.
	if math.Abs(s.bodyZInSensor[2]-1) > 1e-9 {
		t.Fatalf("bodyZ=%v want [0 0 1]", s.bodyZInSensor)
	}
	// Expect X forward remains +X (already horizontal).
	if math.Abs(s.bodyXInSensor[0]-1) > 1e-9 {
		t.Fatalf("bodyX=%v want [1 0 0]", s.bodyXInSensor)
	}
}

func TestApplyOrientationFromGravity_ForwardAxisNearlyVerticalErrors(t *testing.T) {
	s := New(Config{Enable: false})
	s.forwardAxis = 3
	if err := s.applyOrientationFromGravity([3]float64{0, 0, 1}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestApplyOrientationFromGravity_ZeroGravityErrors(t *testing.T) {
	s := New(Config{Enable: false})
	s.forwardAxis = 1
	if err := s.applyOrientationFromGravity([3]float64{0, 0, 0}); err == nil {
		t.Fatalf("expected error")
	}
}
