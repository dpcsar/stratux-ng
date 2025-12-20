package sim

import (
	"math"
	"time"
)

type TrafficTarget struct {
	LatDeg   float64
	LonDeg   float64
	AltFeet  int
	TrackDeg float64
	GroundKt int
}

type TrafficSim struct {
	CenterLatDeg float64
	CenterLonDeg float64
	BaseAltFeet  int
	GroundKt     int
	RadiusNm     float64
	Period       time.Duration
}

// Targets returns N targets orbiting around the configured center.
func (s TrafficSim) Targets(now time.Time, count int) []TrafficTarget {
	if count <= 0 {
		return nil
	}

	period := s.Period
	if period <= 0 {
		period = 90 * time.Second
	}
	radiusNm := s.RadiusNm
	if radiusNm <= 0 {
		radiusNm = 2.0
	}
	groundKt := s.GroundKt
	if groundKt <= 0 {
		groundKt = 120
	}

	// Convert NM to degrees latitude (~60 NM per degree).
	radiusDeg := radiusNm / 60.0

	phase := float64(now.UnixNano()%period.Nanoseconds()) / float64(period.Nanoseconds())
	baseTheta := 2 * math.Pi * phase

	out := make([]TrafficTarget, 0, count)
	for i := 0; i < count; i++ {
		offset := 2 * math.Pi * (float64(i) / float64(count))
		theta := baseTheta + offset

		latDeg := s.CenterLatDeg + radiusDeg*math.Cos(theta)
		lonDeg := s.CenterLonDeg + radiusDeg*math.Sin(theta)/math.Cos(s.CenterLatDeg*math.Pi/180.0)
		trk := math.Mod((theta*180/math.Pi)+90, 360)

		alt := s.BaseAltFeet
		if alt == 0 {
			alt = 4500
		}
		// Stagger altitude a little between targets.
		alt += (i - count/2) * 300

		out = append(out, TrafficTarget{
			LatDeg:   latDeg,
			LonDeg:   lonDeg,
			AltFeet:  alt,
			TrackDeg: trk,
			GroundKt: groundKt,
		})
	}

	return out
}
