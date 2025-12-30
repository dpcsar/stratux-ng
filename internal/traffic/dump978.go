package traffic

import (
	"encoding/json"
	"math"
	"strings"

	"stratux-ng/internal/gdl90"
)

// ParseDump978NDJSON parses a single NDJSON line from dump978-fa's --json-port
// output and returns a unified traffic update (position + metadata) when
// possible.
//
// dump978-fa emits one JSON object per downlink message (no wrapper array).
// This parser is intentionally tolerant: unknown fields are ignored and parse
// failures are reported via false so the stream stays healthy.
func ParseDump978NDJSON(raw json.RawMessage) (TrafficUpdate, bool) {
	var msg dump978Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return TrafficUpdate{}, false
	}
	return msg.toUpdate()
}

type dump978Message struct {
	Address string `json:"address"`

	Position *struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"position"`

	PressureAltitude  *int     `json:"pressure_altitude"`
	GeometricAltitude *int     `json:"geometric_altitude"`
	NIC               *int     `json:"nic"`
	NACp              *int     `json:"nac_p"`
	GroundSpeed       *float64 `json:"ground_speed"`
	TrueTrack         *float64 `json:"true_track"`
	VerticalVelBaro   *int     `json:"vertical_velocity_barometric"`
	VerticalVelGeom   *int     `json:"vertical_velocity_geometric"`
	AirGroundState    *string  `json:"airground_state"`
	Callsign          *string  `json:"callsign"`
}

func (m dump978Message) toUpdate() (TrafficUpdate, bool) {
	addr := strings.TrimSpace(m.Address)
	if addr == "" {
		return TrafficUpdate{}, false
	}
	icao, err := gdl90.ParseICAOHex(addr)
	if err != nil {
		return TrafficUpdate{}, false
	}
	if m.Position == nil {
		return TrafficUpdate{}, false
	}
	lat := m.Position.Lat
	lon := m.Position.Lon

	meta := MetadataUpdate{ICAO: icao}
	altFeet := 0
	if m.GeometricAltitude != nil {
		altFeet = *m.GeometricAltitude
	} else if m.PressureAltitude != nil {
		altFeet = *m.PressureAltitude
	}
	if m.GeometricAltitude != nil || m.PressureAltitude != nil {
		meta.AltFeet = altFeet
		meta.HasAlt = true
	}

	groundKt := 0
	if m.GroundSpeed != nil {
		groundKt = int(math.Round(*m.GroundSpeed))
		if groundKt < 0 {
			groundKt = 0
		}
		meta.GroundKt = groundKt
		meta.HasGround = true
	}

	trackDeg := 0.0
	if m.TrueTrack != nil {
		trackDeg = *m.TrueTrack
		meta.TrackDeg = trackDeg
		meta.HasTrack = true
	}

	vvelFpm := 0
	if m.VerticalVelGeom != nil {
		vvelFpm = *m.VerticalVelGeom
	} else if m.VerticalVelBaro != nil {
		vvelFpm = *m.VerticalVelBaro
	}
	if m.VerticalVelGeom != nil || m.VerticalVelBaro != nil {
		meta.VvelFpm = vvelFpm
		meta.HasVvel = true
	}

	onGround := groundKt == 0
	if m.AirGroundState != nil {
		s := strings.TrimSpace(*m.AirGroundState)
		if s == "ground" {
			onGround = true
		} else if s == "airborne" || s == "supersonic" {
			onGround = false
		}
	}
	if m.AirGroundState != nil || m.GroundSpeed != nil {
		meta.OnGround = onGround
		meta.HasOnGround = true
	}

	nic := byte(8)
	if m.NIC != nil {
		v := *m.NIC
		if v < 0 {
			v = 0
		}
		if v > 15 {
			v = 15
		}
		nic = byte(v)
	}

	nacp := byte(8)
	if m.NACp != nil {
		v := *m.NACp
		if v < 0 {
			v = 0
		}
		if v > 15 {
			v = 15
		}
		nacp = byte(v)
	}

	tail := ""
	if m.Callsign != nil {
		tail = strings.TrimSpace(*m.Callsign)
		if len(tail) > 8 {
			tail = tail[:8]
		}
	}
	if tail != "" {
		meta.Tail = tail
		meta.HasTail = true
	}

	traffic := gdl90.Traffic{
		AddrType:        0x00,
		ICAO:            icao,
		LatDeg:          lat,
		LonDeg:          lon,
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
	}

	return TrafficUpdate{
		ICAO:    icao,
		Traffic: &traffic,
		Meta:    meta,
		Source:  Source978,
	}, true
}
