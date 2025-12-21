package sim

import (
	"math"
	"time"
)

type TrafficTarget struct {
	LatDeg       float64
	LonDeg       float64
	AltFeet      int
	TrackDeg     float64
	GroundKt     int
	VvelFpm      int
	Extrapolated bool
	Visible      bool
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
		// Deterministic “dropouts” (targets disappear/reappear) to exercise EFB
		// stale-target handling.
		//
		// Keep the two special targets always visible:
		// - i==0: stationary/on-ground
		// - i==1: straight-line crossing
		visible := true
		if i >= 2 {
			// Only a subset drops out, so the screen isn't empty.
			if i%6 == 0 {
				// Stagger each target's dropout window.
				p := math.Mod(phase+float64(i)*0.07, 1.0)
				// 10% of the cycle invisible.
				if p >= 0.20 && p < 0.30 {
					visible = false
				}
			}
		}

		alt := s.BaseAltFeet
		if alt == 0 {
			alt = 4500
		}
		// Stagger altitude between targets.
		alt += (i - count/2) * 300

		// Small deterministic vertical oscillation per target.
		ampAlt := 150.0
		vp := period
		if vp <= 0 {
			vp = 90 * time.Second
		}
		phaseV := float64((now.UnixNano()+int64(i)*31_000_000)%vp.Nanoseconds()) / float64(vp.Nanoseconds())
		wV := 2 * math.Pi * phaseV
		altF := float64(alt) + ampAlt*math.Sin(wV)
		alt = int(math.Round(altF))
		ftPerSec := ampAlt * (2 * math.Pi / vp.Seconds()) * math.Cos(wV)
		vvelFpm := int(math.Round(ftPerSec * 60))

		// Add a small subset of special behaviors to make the traffic picture
		// more interesting (while remaining deterministic):
		// - i==0: stationary "on-ground" target near center
		// - i==1: straight-line crossing target through center
		// - others: orbiters with alternating direction and radius
		if i == 0 {
			out = append(out, TrafficTarget{
				LatDeg:       s.CenterLatDeg + radiusDeg*0.10,
				LonDeg:       s.CenterLonDeg,
				AltFeet:      alt,
				TrackDeg:     0,
				GroundKt:     0,
				VvelFpm:      0,
				Extrapolated: false,
				Visible:      true,
			})
			continue
		}

		if i == 1 {
			// Cross west->east over the scenario period.
			// x in [-1,1], y fixed small offset.
			x := math.Sin(baseTheta)
			y := -0.15
			latDeg := s.CenterLatDeg + radiusDeg*y
			lonDeg := s.CenterLonDeg + (radiusDeg*x)/math.Cos(s.CenterLatDeg*math.Pi/180.0)
			out = append(out, TrafficTarget{
				LatDeg:       latDeg,
				LonDeg:       lonDeg,
				AltFeet:      alt,
				TrackDeg:     90,
				GroundKt:     groundKt,
				VvelFpm:      vvelFpm,
				Extrapolated: i%7 == 0,
				Visible:      true,
			})
			continue
		}

		offset := 2 * math.Pi * (float64(i) / float64(count))
		dir := 1.0
		if i%2 == 0 {
			dir = -1.0
		}
		theta := dir*baseTheta + offset

		// Vary radius slightly per target within [0.6, 1.0] of radiusDeg.
		rScale := 0.6 + 0.4*math.Mod(float64(i)*0.37, 1.0)
		r := radiusDeg * rScale

		latDeg := s.CenterLatDeg + r*math.Cos(theta)
		lonDeg := s.CenterLonDeg + r*math.Sin(theta)/math.Cos(s.CenterLatDeg*math.Pi/180.0)
		trk := math.Mod((theta*180/math.Pi)+90, 360)
		if dir < 0 {
			trk = math.Mod((theta*180/math.Pi)+270, 360)
		}

		// Small subset with reduced speed to create relative motion variety.
		spd := groundKt
		if i%5 == 0 {
			spd = int(math.Round(float64(groundKt) * 0.5))
		}

		out = append(out, TrafficTarget{
			LatDeg:       latDeg,
			LonDeg:       lonDeg,
			AltFeet:      alt,
			TrackDeg:     trk,
			GroundKt:     spd,
			VvelFpm:      vvelFpm,
			Extrapolated: i%7 == 0,
			Visible:      visible,
		})
	}

	return out
}
