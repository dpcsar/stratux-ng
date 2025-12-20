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
	theta := 2 * math.Pi * phase

	latDeg = s.CenterLatDeg + radiusDeg*math.Cos(theta)
	lonDeg = s.CenterLonDeg + radiusDeg*math.Sin(theta)/math.Cos(s.CenterLatDeg*math.Pi/180.0)

	// Track tangent to the circle (degrees true-ish for sim).
	trackDeg = math.Mod((theta*180/math.Pi)+90, 360)
	return latDeg, lonDeg, trackDeg
}
