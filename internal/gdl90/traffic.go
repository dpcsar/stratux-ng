package gdl90

import "math"

// Traffic represents a single traffic target for GDL90 Traffic Report (0x14).
//
// This is intentionally minimal and tuned for live Stratux-NG traffic updates.
type Traffic struct {
	AddrType        byte // 0 = ADS-B with ICAO address (common)
	ICAO            [3]byte
	LatDeg          float64
	LonDeg          float64
	AltFeet         int
	NIC             byte // 0-15
	NACp            byte // 0-15
	GroundKt        int
	TrackDeg        float64
	VvelFpm         int // ft/min
	OnGround        bool
	Extrapolated    bool
	EmitterCategory byte
	Tail            string // 8 chars max
	PriorityStatus  byte   // upper nibble in msg[27]
}

// TrafficReportFrame builds and frames a GDL90 Traffic Report (0x14).
//
// Packing mirrors Stratux's makeTrafficReportMsg for broad EFB compatibility.
func TrafficReportFrame(t Traffic) []byte {
	msg := make([]byte, 28)
	msg[0] = 0x14

	addrType := t.AddrType
	// NOTE: alert bit (0x10) not modeled in MVP.
	msg[1] = addrType

	msg[2] = t.ICAO[0]
	msg[3] = t.ICAO[1]
	msg[4] = t.ICAO[2]

	lat := encodeLatLon24(t.LatDeg)
	msg[5], msg[6], msg[7] = lat[0], lat[1], lat[2]

	lon := encodeLatLon24(t.LonDeg)
	msg[8], msg[9], msg[10] = lon[0], lon[1], lon[2]

	alt := encodeAltitude12(t.AltFeet)
	msg[11] = byte((alt & 0xFF0) >> 4)
	msg[12] = byte((alt & 0x00F) << 4)

	// m field indicator bits in lower nibble of msg[12].
	// - bit0: track valid (true track)
	// - bit2: extrapolated
	// - bit3: airborne
	msg[12] |= 0x01
	if t.Extrapolated {
		msg[12] |= 0x04
	}
	if !t.OnGround {
		msg[12] |= 0x08
	}

	nic := t.NIC & 0x0F
	nacp := t.NACp & 0x0F
	msg[13] = ((nic << 4) & 0xF0) | nacp

	spd := encodeU12(t.GroundKt)
	msg[14] = byte((spd & 0x0FF0) >> 4)
	msg[15] = byte((spd & 0x000F) << 4)

	vv64 := int32(math.Round(float64(t.VvelFpm) / 64.0))
	vv64 = clampI32(vv64, -2047, 2047)
	vvel := int16(vv64)
	vvelU := uint16(vvel) & 0x0FFF
	msg[15] |= byte((vvelU & 0x0F00) >> 8)
	msg[16] = byte(vvelU & 0x00FF)

	msg[17] = encodeTrack8(t.TrackDeg)

	emitter := t.EmitterCategory
	if emitter == 0 {
		emitter = 0x01
	}
	msg[18] = emitter

	tail := sanitizeCallsign(t.Tail)
	copy(msg[19:27], []byte(tail))

	msg[27] = (t.PriorityStatus & 0x0F) << 4

	return Frame(msg)
}
