package traffic

import (
	"encoding/json"
	"math"
	"strings"

	"stratux-ng/internal/gdl90"
)

// MetadataUpdate captures supplemental fields emitted by dump1090 even when a
// message does not include a full position.
type MetadataUpdate struct {
	ICAO [3]byte

	Tail        string
	HasTail     bool
	GroundKt    int
	HasGround   bool
	TrackDeg    float64
	HasTrack    bool
	VvelFpm     int
	HasVvel     bool
	AltFeet     int
	HasAlt      bool
	OnGround    bool
	HasOnGround bool
	Squawk      string
	HasSquawk   bool
}

// Empty reports whether the update contains any useful metadata.
func (m MetadataUpdate) Empty() bool {
	return !m.HasTail && !m.HasGround && !m.HasTrack && !m.HasVvel && !m.HasAlt && !m.HasOnGround && !m.HasSquawk
}

// ParseDump1090RawJSON parses a single line from dump1090's Stratux NDJSON
// stream. Position-bearing messages populate Traffic so downstream consumers
// can emit GDL90. Metadata-only messages return Meta updates so callers can
// augment existing traffic without producing new GDL90 frames.
func ParseDump1090RawJSON(raw json.RawMessage) (TrafficUpdate, bool) {
	var msg dump1090RawMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return TrafficUpdate{}, false
	}
	addr, ok := normalizeICAO(msg.IcaoAddr)
	if !ok {
		return TrafficUpdate{}, false
	}
	icao, ok := icaoBytes(addr)
	if !ok {
		return TrafficUpdate{}, false
	}

	out := TrafficUpdate{
		ICAO: icao,
		Meta: MetadataUpdate{ICAO: icao},
	}
	out.Source = Source1090

	if msg.Tail != nil {
		tail := strings.ToUpper(strings.TrimSpace(*msg.Tail))
		if tail != "" {
			out.Meta.Tail = tail
			out.Meta.HasTail = true
		}
	}

	if msg.Squawk != nil {
		sq := strings.TrimSpace(*msg.Squawk)
		if sq != "" {
			out.Meta.Squawk = sq
			out.Meta.HasSquawk = true
		}
	}

	if msg.Alt != nil {
		out.Meta.AltFeet = *msg.Alt
		out.Meta.HasAlt = true
	}

	groundKt := 0
	if msg.SpeedValid && msg.Speed != nil {
		groundKt = clampNonNegative(int(math.Round(*msg.Speed)))
		out.Meta.GroundKt = groundKt
		out.Meta.HasGround = true
	}

	if msg.SpeedValid && msg.Track != nil {
		out.Meta.TrackDeg = *msg.Track
		out.Meta.HasTrack = true
	}

	if msg.Vvel != nil {
		out.Meta.VvelFpm = *msg.Vvel
		out.Meta.HasVvel = true
	}

	onGround := false
	if msg.OnGround != nil {
		onGround = *msg.OnGround
		out.Meta.OnGround = onGround
		out.Meta.HasOnGround = true
	} else if msg.SpeedValid && msg.Speed != nil {
		onGround = groundKt == 0
		out.Meta.OnGround = onGround
		out.Meta.HasOnGround = true
	}

	if msg.PositionValid && msg.Lat != nil && msg.Lng != nil {
		altFeet := 0
		if msg.Alt != nil {
			altFeet = *msg.Alt
		}

		trackDeg := 0.0
		if msg.SpeedValid && msg.Track != nil {
			trackDeg = *msg.Track
		}

		vvel := 0
		if msg.Vvel != nil {
			vvel = *msg.Vvel
		}

		nacp := byte(8)
		if msg.NACp != nil {
			nacp = clampNibble(*msg.NACp)
			if nacp == 0 {
				nacp = 8
			}
		}

		nic := deriveNIC(msg.DF, msg.TypeCode, msg.SubtypeCode)
		if nic == 0 {
			nic = 8
		}
		if nacp < 7 && nacp < nic {
			nacp = nic
		}

		emitter := byte(0x01)
		if msg.EmitterCategory != nil {
			emitter = byte(clampNonNegative(*msg.EmitterCategory))
			if emitter == 0 {
				emitter = 0x01
			}
		}

		tail := ""
		if msg.Tail != nil {
			tail = strings.ToUpper(strings.TrimSpace(*msg.Tail))
		}

		t := gdl90.Traffic{
			AddrType:        addrType(msg.DF, msg.CA),
			ICAO:            icao,
			LatDeg:          *msg.Lat,
			LonDeg:          *msg.Lng,
			AltFeet:         altFeet,
			NIC:             nic,
			NACp:            nacp,
			GroundKt:        groundKt,
			TrackDeg:        trackDeg,
			VvelFpm:         vvel,
			OnGround:        onGround,
			Extrapolated:    false,
			EmitterCategory: emitter,
			Tail:            tail,
			PriorityStatus:  0,
		}
		out.Traffic = &t
	}

	if out.Traffic == nil && out.Meta.Empty() {
		return TrafficUpdate{}, false
	}

	return out, true
}

type dump1090RawMessage struct {
	IcaoAddr        uint32   `json:"Icao_addr"`
	DF              int      `json:"DF"`
	CA              int      `json:"CA"`
	TypeCode        int      `json:"TypeCode"`
	SubtypeCode     int      `json:"SubtypeCode"`
	PositionValid   bool     `json:"Position_valid"`
	Lat             *float64 `json:"Lat"`
	Lng             *float64 `json:"Lng"`
	Alt             *int     `json:"Alt"`
	NACp            *int     `json:"NACp"`
	SpeedValid      bool     `json:"Speed_valid"`
	Speed           *float64 `json:"Speed"`
	Track           *float64 `json:"Track"`
	Vvel            *int     `json:"Vvel"`
	OnGround        *bool    `json:"OnGround"`
	Tail            *string  `json:"Tail"`
	Squawk          *string  `json:"Squawk"`
	EmitterCategory *int     `json:"Emitter_category"`
}

func normalizeICAO(addr uint32) (uint32, bool) {
	if addr == 0 || addr == 0x07FFFFFF {
		return 0, false
	}
	if addr&0x01000000 != 0 {
		addr &= 0x00FFFFFF
	}
	addr &= 0x00FFFFFF
	if addr == 0 {
		return 0, false
	}
	return addr, true
}

func icaoBytes(addr uint32) ([3]byte, bool) {
	var out [3]byte
	if addr > 0xFFFFFF {
		return out, false
	}
	out[0] = byte((addr >> 16) & 0xFF)
	out[1] = byte((addr >> 8) & 0xFF)
	out[2] = byte(addr & 0xFF)
	return out, true
}

func clampNonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func clampNibble(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 15 {
		return 15
	}
	return byte(v)
}

func addrType(df, ca int) byte {
	if df != 18 {
		return 0
	}
	switch ca {
	case 6:
		return 2
	case 2:
		return 2
	case 5:
		return 3
	default:
		return 0
	}
}

func deriveNIC(df, typeCode, subtype int) byte {
	if (df != 17 && df != 18) || typeCode < 5 || typeCode > 22 || typeCode == 19 {
		return 0
	}
	nic := 0
	switch typeCode {
	case 0, 8, 18, 22:
		nic = 0
	case 17:
		nic = 1
	case 16:
		if subtype == 1 {
			nic = 3
		} else {
			nic = 2
		}
	case 15:
		nic = 4
	case 14:
		nic = 5
	case 13:
		nic = 6
	case 12:
		nic = 7
	case 11:
		if subtype == 1 {
			nic = 9
		} else {
			nic = 8
		}
	case 10, 21:
		nic = 10
	case 9, 20:
		nic = 11
	}
	if nic < 0 {
		nic = 0
	}
	if nic > 15 {
		nic = 15
	}
	return byte(nic)
}
