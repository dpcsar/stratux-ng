package traffic

import (
	"encoding/json"
	"math"
	"strings"

	"stratux-ng/internal/gdl90"
)

// ParseAircraftJSON parses a JSON object that contains aircraft data (e.g.
// dump1090-fa aircraft.json) and returns any aircraft targets contained within.
//
// It is intentionally tolerant: unknown fields are ignored and parse failures
// return an empty slice (not an error) so the stream stays healthy.
func ParseAircraftJSON(raw json.RawMessage) []gdl90.Traffic {
	// Either:
	// 1) {"aircraft":[{...},{...}]}
	// 2) {"hex":"ABC123", ...}
	var wrap struct {
		Aircraft []aircraftJSONAircraft `json:"aircraft"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	if len(wrap.Aircraft) > 0 {
		out := make([]gdl90.Traffic, 0, len(wrap.Aircraft))
		for _, a := range wrap.Aircraft {
			if t, ok := a.toTraffic(); ok {
				out = append(out, t)
			}
		}
		return out
	}

	var single aircraftJSONAircraft
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil
	}
	if t, ok := single.toTraffic(); ok {
		return []gdl90.Traffic{t}
	}
	return nil
}

type aircraftJSONAircraft struct {
	Hex      string   `json:"hex"`
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
	AltBaro  *int     `json:"alt_baro"`
	AltGeom  *int     `json:"alt_geom"`
	GS       *float64 `json:"gs"`
	Track    *float64 `json:"track"`
	BaroRate *int     `json:"baro_rate"`
	GeomRate *int     `json:"geom_rate"`
	Flight   string   `json:"flight"`
	NIC      *int     `json:"nic"`
	NACp     *int     `json:"nac_p"`
	Ground   *bool    `json:"ground"`
	GND      *bool    `json:"gnd"`
}

func (a aircraftJSONAircraft) toTraffic() (gdl90.Traffic, bool) {
	hex := strings.TrimSpace(a.Hex)
	if hex == "" {
		return gdl90.Traffic{}, false
	}
	icao, err := gdl90.ParseICAOHex(hex)
	if err != nil {
		return gdl90.Traffic{}, false
	}
	if a.Lat == nil || a.Lon == nil {
		// Can't build a usable traffic target without a position.
		return gdl90.Traffic{}, false
	}

	altFeet := 0
	if a.AltGeom != nil {
		altFeet = *a.AltGeom
	} else if a.AltBaro != nil {
		altFeet = *a.AltBaro
	}

	groundKt := 0
	if a.GS != nil {
		groundKt = int(math.Round(*a.GS))
		if groundKt < 0 {
			groundKt = 0
		}
	}

	trackDeg := 0.0
	if a.Track != nil {
		trackDeg = *a.Track
	}

	vvelFpm := 0
	if a.GeomRate != nil {
		vvelFpm = *a.GeomRate
	} else if a.BaroRate != nil {
		vvelFpm = *a.BaroRate
	}

	onGround := groundKt == 0
	if a.Ground != nil {
		onGround = *a.Ground
	} else if a.GND != nil {
		onGround = *a.GND
	}

	nic := byte(8)
	if a.NIC != nil {
		v := *a.NIC
		if v < 0 {
			v = 0
		}
		if v > 15 {
			v = 15
		}
		nic = byte(v)
	}
	nacp := byte(8)
	if a.NACp != nil {
		v := *a.NACp
		if v < 0 {
			v = 0
		}
		if v > 15 {
			v = 15
		}
		nacp = byte(v)
	}

	tail := strings.TrimSpace(a.Flight)
	if len(tail) > 8 {
		tail = tail[:8]
	}

	return gdl90.Traffic{
		AddrType:        0x00,
		ICAO:            icao,
		LatDeg:          *a.Lat,
		LonDeg:          *a.Lon,
		AltFeet:         altFeet,
		NIC:             nic,
		NACp:            nacp,
		GroundKt:        groundKt,
		TrackDeg:        trackDeg,
		VvelFpm:         vvelFpm,
		OnGround:        onGround,
		Extrapolated:    false,
		EmitterCategory: 0x01,
		Tail:            tail,
		PriorityStatus:  0,
	}, true
}
