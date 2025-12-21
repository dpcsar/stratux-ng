package sim

import (
	"math"
	"time"
)

type OwnshipSim struct {
	CenterLatDeg float64
	CenterLonDeg float64
	AltFeet      int
	GroundKt     int
	RadiusNm     float64
	Period       time.Duration
}

// Kinematics returns deterministic position plus a simple vertical profile.
//
// Altitude is a sinusoid around AltFeet, and vertical speed is its derivative.
func (s OwnshipSim) Kinematics(now time.Time) (latDeg, lonDeg, trackDeg float64, altFeet int, vvelFpm int) {
	latDeg, lonDeg, trackDeg = s.Position(now)

	baseAlt := s.AltFeet
	if baseAlt == 0 {
		baseAlt = 3000
	}
	period := s.Period
	if period <= 0 {
		period = 120 * time.Second
	}
	// Vertical period is decoupled from horizontal to avoid repetitive sync.
	vp := period / 2
	if vp < 30*time.Second {
		vp = 30 * time.Second
	}
	amp := 500.0 // ft

	phase := float64(now.UnixNano()%vp.Nanoseconds()) / float64(vp.Nanoseconds())
	w := 2 * math.Pi * phase

	alt := float64(baseAlt) + amp*math.Sin(w)
	altFeet = int(math.Round(alt))

	// d/dt (amp*sin(w)) where w = 2πt/T => amp*(2π/T)*cos(w)
	ftPerSec := amp * (2 * math.Pi / vp.Seconds()) * math.Cos(w)
	vvelFpm = int(math.Round(ftPerSec * 60))
	return latDeg, lonDeg, trackDeg, altFeet, vvelFpm
}

// Position returns a simple circular track around the configured center.
func (s OwnshipSim) Position(now time.Time) (latDeg, lonDeg, trackDeg float64) {
	period := s.Period
	if period <= 0 {
		period = 120 * time.Second
	}
	radiusNm := s.RadiusNm
	if radiusNm <= 0 {
		radiusNm = 0.5
	}

	// Convert NM to degrees latitude (~60 NM per degree).
	radiusDeg := radiusNm / 60.0

	phase := float64(now.UnixNano()%period.Nanoseconds()) / float64(period.Nanoseconds())

	// More interesting than a pure circle: a deterministic figure-eight (Lissajous)
	// path that stays within the configured radius.
	//
	// x: east-west component (scaled by cos(lat) for lon degrees)
	// y: north-south component
	//	  x = cos(2πt)
	//	  y = 0.5*sin(4πt)
	//
	// y is kept within [-0.5, 0.5] so we remain comfortably within the
	// radius bounds used by tests.
	w := 2 * math.Pi * phase
	x := math.Cos(w)
	y := 0.5 * math.Sin(2*w)

	latDeg = s.CenterLatDeg + radiusDeg*y
	lonDeg = s.CenterLonDeg + (radiusDeg*x)/math.Cos(s.CenterLatDeg*math.Pi/180.0)

	// Track based on instantaneous velocity (atan2(east, north)).
	vx := -2 * math.Pi * math.Sin(w)
	vy := 2 * math.Pi * math.Cos(2*w)
	trackRad := math.Atan2(vx, vy)
	trackDeg = math.Mod((trackRad*180/math.Pi)+360, 360)
	return latDeg, lonDeg, trackDeg
}
