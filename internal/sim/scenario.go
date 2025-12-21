package sim

import (
	"fmt"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// ScenarioScript is a deterministic, script-driven simulation description.
//
// Time is expressed as Go duration strings (e.g. "0s", "250ms", "10s").
// If Duration is zero, it is derived from the latest keyframe time.
//
// Fields are intentionally minimal; this is meant to reproduce EFB behaviors
// precisely without adding extra scope.
//
// YAML schema (v1):
//
//	version: 1
//	duration: 30s
//	ownship:
//	  icao: "F00000"
//	  callsign: "STRATUX"
//	  gps_horizontal_accuracy_m: 50
//	  keyframes:
//	    - t: 0s
//	      lat_deg: 45.0
//	      lon_deg: -122.0
//	      alt_feet: 3000
//	      ground_kt: 90
//	      track_deg: 90
//	traffic:
//	  - icao: "F10001"
//	    callsign: "TGT0001"
//	    keyframes: ...
//
// All keyframes for ownship and traffic targets must be sorted by time and use
// non-decreasing t values.
//
// Note: ICAO/callsign strings are parsed/validated at higher layers that know
// about GDL90 constraints.
//
// This type is unmarshaled with yaml.v3 in LoadScenarioScript.
//
// Keep this struct stable: scripts are test fixtures.
//
//nolint:revive // exported for YAML, but used primarily internally
type ScenarioScript struct {
	Version  int               `yaml:"version"`
	Duration time.Duration     `yaml:"duration"`
	Ownship  ScenarioOwnship   `yaml:"ownship"`
	Traffic  []ScenarioTraffic `yaml:"traffic"`
}

// ScenarioOwnship describes ownship keyframes.
//
// gps_horizontal_accuracy_m is used to derive NACp similarly to Stratux.
//
//nolint:revive
type ScenarioOwnship struct {
	ICAO                   string            `yaml:"icao"`
	Callsign               string            `yaml:"callsign"`
	GPSHorizontalAccuracyM float64           `yaml:"gps_horizontal_accuracy_m"`
	Keyframes              []OwnshipKeyframe `yaml:"keyframes"`
}

// OwnshipKeyframe is a time-stamped ownship state.
//
//nolint:revive
type OwnshipKeyframe struct {
	T        time.Duration `yaml:"t"`
	LatDeg   float64       `yaml:"lat_deg"`
	LonDeg   float64       `yaml:"lon_deg"`
	AltFeet  int           `yaml:"alt_feet"`
	GroundKt int           `yaml:"ground_kt"`
	TrackDeg float64       `yaml:"track_deg"`
}

// ScenarioTraffic describes a single traffic target timeline.
//
//nolint:revive
type ScenarioTraffic struct {
	ICAO      string            `yaml:"icao"`
	Callsign  string            `yaml:"callsign"`
	Keyframes []TrafficKeyframe `yaml:"keyframes"`
}

// TrafficKeyframe is a time-stamped traffic state.
//
//nolint:revive
type TrafficKeyframe struct {
	T        time.Duration `yaml:"t"`
	LatDeg   float64       `yaml:"lat_deg"`
	LonDeg   float64       `yaml:"lon_deg"`
	AltFeet  int           `yaml:"alt_feet"`
	GroundKt int           `yaml:"ground_kt"`
	TrackDeg float64       `yaml:"track_deg"`
}

// Scenario is the validated, runtime representation.
//
// Use StateAt to compute the deterministic state at a given elapsed time.
//
//nolint:revive
type Scenario struct {
	script ScenarioScript
	// Derived duration (script.Duration or max keyframe time).
	duration time.Duration
}

// LoadScenarioScript reads and unmarshals a YAML scenario script from path.
func LoadScenarioScript(path string) (ScenarioScript, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ScenarioScript{}, err
	}
	return ParseScenarioScriptYAML(b)
}

// ParseScenarioScriptYAML parses a YAML scenario script.
func ParseScenarioScriptYAML(b []byte) (ScenarioScript, error) {
	var s ScenarioScript
	if err := yaml.Unmarshal(b, &s); err != nil {
		return ScenarioScript{}, err
	}
	return s, nil
}

// NewScenario validates script and returns a runtime Scenario.
func NewScenario(script ScenarioScript) (*Scenario, error) {
	if script.Version == 0 {
		script.Version = 1
	}
	if script.Version != 1 {
		return nil, fmt.Errorf("unsupported scenario version %d", script.Version)
	}
	if len(script.Ownship.Keyframes) == 0 {
		return nil, fmt.Errorf("ownship.keyframes is required")
	}
	if err := validateNonDecreasingOwnship(script.Ownship.Keyframes); err != nil {
		return nil, err
	}

	for i := range script.Traffic {
		if len(script.Traffic[i].Keyframes) == 0 {
			return nil, fmt.Errorf("traffic[%d].keyframes is required", i)
		}
		if err := validateNonDecreasingTraffic(script.Traffic[i].Keyframes, i); err != nil {
			return nil, err
		}
	}

	dur := script.Duration
	if dur <= 0 {
		dur = maxKeyframeTime(script)
	}
	if dur <= 0 {
		return nil, fmt.Errorf("duration is required (or deriveable from keyframes)")
	}

	return &Scenario{script: script, duration: dur}, nil
}

// Duration returns the effective scenario duration.
func (s *Scenario) Duration() time.Duration {
	if s == nil {
		return 0
	}
	return s.duration
}

// ScenarioOwnshipState is the computed ownship state at a time.
//
//nolint:revive
type ScenarioOwnshipState struct {
	ICAO                   string
	Callsign               string
	GPSHorizontalAccuracyM float64
	LatDeg                 float64
	LonDeg                 float64
	AltFeet                int
	GroundKt               int
	TrackDeg               float64
}

// ScenarioTrafficState is the computed traffic state at a time.
//
//nolint:revive
type ScenarioTrafficState struct {
	ICAO     string
	Callsign string
	LatDeg   float64
	LonDeg   float64
	AltFeet  int
	GroundKt int
	TrackDeg float64
	Index    int
}

// ScenarioState is the computed scenario state at a time.
//
//nolint:revive
type ScenarioState struct {
	Ownship ScenarioOwnshipState
	Traffic []ScenarioTrafficState
}

// StateAt computes scenario state at elapsed.
//
// If loop is true, elapsed wraps around Duration(). Otherwise elapsed is clamped
// to [0, Duration()].
func (s *Scenario) StateAt(elapsed time.Duration, loop bool) ScenarioState {
	if s == nil {
		return ScenarioState{}
	}
	if elapsed < 0 {
		elapsed = 0
	}
	if s.duration > 0 {
		if loop {
			elapsed = elapsed % s.duration
		} else if elapsed > s.duration {
			elapsed = s.duration
		}
	}

	own := sampleOwnship(s.script.Ownship, elapsed)
	out := ScenarioState{Ownship: own}
	if len(s.script.Traffic) == 0 {
		return out
	}

	out.Traffic = make([]ScenarioTrafficState, 0, len(s.script.Traffic))
	for i := range s.script.Traffic {
		t := sampleTraffic(s.script.Traffic[i], elapsed)
		t.Index = i
		out.Traffic = append(out.Traffic, t)
	}
	return out
}

func validateNonDecreasingOwnship(kfs []OwnshipKeyframe) error {
	for i := range kfs {
		if kfs[i].T < 0 {
			return fmt.Errorf("ownship.keyframes[%d].t must be >= 0", i)
		}
		if i > 0 && kfs[i].T < kfs[i-1].T {
			return fmt.Errorf("ownship.keyframes must be sorted by t (index %d)", i)
		}
	}
	return nil
}

func validateNonDecreasingTraffic(kfs []TrafficKeyframe, ti int) error {
	for i := range kfs {
		if kfs[i].T < 0 {
			return fmt.Errorf("traffic[%d].keyframes[%d].t must be >= 0", ti, i)
		}
		if i > 0 && kfs[i].T < kfs[i-1].T {
			return fmt.Errorf("traffic[%d].keyframes must be sorted by t (index %d)", ti, i)
		}
	}
	return nil
}

func maxKeyframeTime(s ScenarioScript) time.Duration {
	max := time.Duration(0)
	for _, kf := range s.Ownship.Keyframes {
		if kf.T > max {
			max = kf.T
		}
	}
	for _, tr := range s.Traffic {
		for _, kf := range tr.Keyframes {
			if kf.T > max {
				max = kf.T
			}
		}
	}
	return max
}

func sampleOwnship(o ScenarioOwnship, t time.Duration) ScenarioOwnshipState {
	kf0, kf1, alpha := selectSegmentOwnship(o.Keyframes, t)

	lat := lerp(kf0.LatDeg, kf1.LatDeg, alpha)
	lon := lerp(kf0.LonDeg, kf1.LonDeg, alpha)
	trk := lerpAngleDeg(kf0.TrackDeg, kf1.TrackDeg, alpha)
	alt := int(lerp(float64(kf0.AltFeet), float64(kf1.AltFeet), alpha))
	gs := int(lerp(float64(kf0.GroundKt), float64(kf1.GroundKt), alpha))

	acc := o.GPSHorizontalAccuracyM
	if acc == 0 {
		acc = 50
	}

	return ScenarioOwnshipState{
		ICAO:                   o.ICAO,
		Callsign:               o.Callsign,
		GPSHorizontalAccuracyM: acc,
		LatDeg:                 lat,
		LonDeg:                 lon,
		AltFeet:                alt,
		GroundKt:               gs,
		TrackDeg:               trk,
	}
}

func sampleTraffic(tr ScenarioTraffic, t time.Duration) ScenarioTrafficState {
	kf0, kf1, alpha := selectSegmentTraffic(tr.Keyframes, t)

	lat := lerp(kf0.LatDeg, kf1.LatDeg, alpha)
	lon := lerp(kf0.LonDeg, kf1.LonDeg, alpha)
	trk := lerpAngleDeg(kf0.TrackDeg, kf1.TrackDeg, alpha)
	alt := int(lerp(float64(kf0.AltFeet), float64(kf1.AltFeet), alpha))
	gs := int(lerp(float64(kf0.GroundKt), float64(kf1.GroundKt), alpha))

	return ScenarioTrafficState{
		ICAO:     tr.ICAO,
		Callsign: tr.Callsign,
		LatDeg:   lat,
		LonDeg:   lon,
		AltFeet:  alt,
		GroundKt: gs,
		TrackDeg: trk,
	}
}

func selectSegmentOwnship(kfs []OwnshipKeyframe, t time.Duration) (OwnshipKeyframe, OwnshipKeyframe, float64) {
	if len(kfs) == 1 {
		return kfs[0], kfs[0], 0
	}
	idx := sort.Search(len(kfs), func(i int) bool { return kfs[i].T > t })
	if idx <= 0 {
		return kfs[0], kfs[0], 0
	}
	if idx >= len(kfs) {
		last := kfs[len(kfs)-1]
		return last, last, 0
	}
	k0 := kfs[idx-1]
	k1 := kfs[idx]
	dt := k1.T - k0.T
	if dt <= 0 {
		return k1, k1, 0
	}
	alpha := float64(t-k0.T) / float64(dt)
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return k0, k1, alpha
}

func selectSegmentTraffic(kfs []TrafficKeyframe, t time.Duration) (TrafficKeyframe, TrafficKeyframe, float64) {
	if len(kfs) == 1 {
		return kfs[0], kfs[0], 0
	}
	idx := sort.Search(len(kfs), func(i int) bool { return kfs[i].T > t })
	if idx <= 0 {
		return kfs[0], kfs[0], 0
	}
	if idx >= len(kfs) {
		last := kfs[len(kfs)-1]
		return last, last, 0
	}
	k0 := kfs[idx-1]
	k1 := kfs[idx]
	dt := k1.T - k0.T
	if dt <= 0 {
		return k1, k1, 0
	}
	alpha := float64(t-k0.T) / float64(dt)
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return k0, k1, alpha
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func lerpAngleDeg(a0, a1, t float64) float64 {
	// Shortest-path interpolation across wraparound.
	// Normalize to [0, 360).
	norm := func(x float64) float64 {
		for x < 0 {
			x += 360
		}
		for x >= 360 {
			x -= 360
		}
		return x
	}
	a0 = norm(a0)
	a1 = norm(a1)
	delta := a1 - a0
	if delta > 180 {
		delta -= 360
	} else if delta < -180 {
		delta += 360
	}
	return norm(a0 + delta*t)
}
