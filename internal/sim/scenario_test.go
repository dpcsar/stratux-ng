package sim

import (
	"testing"
	"time"
)

func TestScenario_ParseAndInterpolateAngleWrap(t *testing.T) {
	yaml := []byte(`
version: 1
# duration derived from last keyframe
ownship:
  icao: "F00000"
  callsign: "STRATUX"
  gps_horizontal_accuracy_m: 50
  keyframes:
    - t: 0s
      lat_deg: 0
      lon_deg: 0
      alt_feet: 0
      ground_kt: 100
      track_deg: 350
    - t: 10s
      lat_deg: 10
      lon_deg: 20
      alt_feet: 1000
      ground_kt: 200
      track_deg: 10
`)

	script, err := ParseScenarioScriptYAML(yaml)
	if err != nil {
		t.Fatalf("ParseScenarioScriptYAML: %v", err)
	}
	scn, err := NewScenario(script)
	if err != nil {
		t.Fatalf("NewScenario: %v", err)
	}
	if scn.Duration() != 10*time.Second {
		t.Fatalf("duration: got %s want %s", scn.Duration(), 10*time.Second)
	}

	st := scn.StateAt(5*time.Second, false)
	// Track 350->10 should interpolate via +20deg shortest path:
	// halfway is 0 degrees.
	if st.Ownship.TrackDeg != 0 {
		t.Fatalf("track wrap interpolation: got %v want 0", st.Ownship.TrackDeg)
	}
	if st.Ownship.LatDeg != 5 {
		t.Fatalf("lat interpolation: got %v want 5", st.Ownship.LatDeg)
	}
	if st.Ownship.LonDeg != 10 {
		t.Fatalf("lon interpolation: got %v want 10", st.Ownship.LonDeg)
	}
	if st.Ownship.AltFeet != 500 {
		t.Fatalf("alt interpolation: got %v want 500", st.Ownship.AltFeet)
	}
	if st.Ownship.GroundKt != 150 {
		t.Fatalf("gs interpolation: got %v want 150", st.Ownship.GroundKt)
	}
}

func TestScenario_LoopAndClamp(t *testing.T) {
	yaml := []byte(`
version: 1
duration: 10s
ownship:
  keyframes:
    - t: 0s
      lat_deg: 0
      lon_deg: 0
      alt_feet: 0
      ground_kt: 0
      track_deg: 0
    - t: 10s
      lat_deg: 10
      lon_deg: 0
      alt_feet: 0
      ground_kt: 0
      track_deg: 0
`)

	script, err := ParseScenarioScriptYAML(yaml)
	if err != nil {
		t.Fatalf("ParseScenarioScriptYAML: %v", err)
	}
	scn, err := NewScenario(script)
	if err != nil {
		t.Fatalf("NewScenario: %v", err)
	}

	// Clamp (no loop): 11s -> end state.
	st := scn.StateAt(11*time.Second, false)
	if st.Ownship.LatDeg != 10 {
		t.Fatalf("clamp lat: got %v want 10", st.Ownship.LatDeg)
	}

	// Loop: 11s -> 1s.
	st2 := scn.StateAt(11*time.Second, true)
	if st2.Ownship.LatDeg != 1 {
		t.Fatalf("loop lat: got %v want 1", st2.Ownship.LatDeg)
	}
}
