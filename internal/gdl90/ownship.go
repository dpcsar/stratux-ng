package gdl90

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

const (
	latLonResolution = 180.0 / 8388608.0 // degrees per LSB for signed 24-bit
	trackResolution  = 360.0 / 256.0
)

type Ownship struct {
	ICAO        [3]byte
	LatDeg      float64
	LonDeg      float64
	AltFeet     int
	HaveNICNACp bool
	NIC         byte // 0-15 (high nibble in msg[13])
	NACp        byte // 0-15 (low nibble in msg[13])
	GroundKt    int
	TrackDeg    float64
	OnGround    bool
	VvelFpm     int
	VvelValid   bool
	Callsign    string
	Emitter     byte // e.g. 0x01 "Light"
	Emergency   byte // upper nibble of msg[27]
}

// OwnshipReportFrame builds and frames a minimal Ownship Report (0x0A).
//
// This is intended for bring-up and simulator use. Fields not yet modeled are
// encoded as "unknown" where applicable.
func OwnshipReportFrame(o Ownship) []byte {
	msg := make([]byte, 28)
	msg[0] = 0x0A

	// Address/alert/type byte.
	// Upper nibble: alert status (0 = no alert)
	// Lower nibble: address type/traffic type (0 = ADS-B with ICAO address)
	msg[1] = 0x00

	msg[2] = o.ICAO[0]
	msg[3] = o.ICAO[1]
	msg[4] = o.ICAO[2]

	lat := encodeLatLon24(o.LatDeg)
	msg[5], msg[6], msg[7] = lat[0], lat[1], lat[2]

	lon := encodeLatLon24(o.LonDeg)
	msg[8], msg[9], msg[10] = lon[0], lon[1], lon[2]

	alt := encodeAltitude12(o.AltFeet)
	msg[11] = byte((alt >> 4) & 0xFF)
	msg[12] = byte((alt & 0x0F) << 4)

	// Misc/flags nibble (lower 4 bits of msg[12]).
	// - bit0: track valid (true track)
	// - bit3: airborne
	msg[12] |= 0x01
	if !o.OnGround {
		msg[12] |= 0x08
	}

	// Position containment / navigational accuracy (msg[13]):
	// high nibble = NIC, low nibble = NACp.
	// Stratux uses NIC=8 and NACp derived from GPS accuracy.
	if o.HaveNICNACp {
		nic := o.NIC & 0x0F
		nacp := o.NACp & 0x0F
		msg[13] = ((nic << 4) & 0xF0) | nacp
	} else {
		msg[13] = 0x80 | 0x08
	}

	// Ground speed (12-bit). We use 1 kt resolution.
	gs := encodeU12(o.GroundKt)
	msg[14] = byte((gs & 0xFF0) >> 4)
	msg[15] = byte((gs & 0x00F) << 4)

	// Vertical velocity (12-bit signed, 64 fpm resolution). 0x800 = unknown.
	vvel := uint16(0x800)
	if o.VvelValid {
		vv := int16(math.Round(float64(o.VvelFpm) / 64.0))
		vvel = uint16(vv) & 0x0FFF
	}
	msg[15] |= byte((vvel & 0x0F00) >> 8)
	msg[16] = byte(vvel & 0x00FF)

	trk := encodeTrack8(o.TrackDeg)
	msg[17] = trk

	emitter := o.Emitter
	if emitter == 0 {
		emitter = 0x01
	}
	msg[18] = emitter

	callsign := sanitizeCallsign(o.Callsign)
	copy(msg[19:27], []byte(callsign))

	msg[27] = (o.Emergency & 0x0F) << 4

	return Frame(msg)
}

func ParseICAOHex(s string) ([3]byte, error) {
	var out [3]byte
	s = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(s), "0x"))
	if len(s) != 6 {
		return out, fmt.Errorf("icao must be 6 hex chars")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return out, err
	}
	copy(out[:], b)
	return out, nil
}

func encodeLatLon24(deg float64) [3]byte {
	v := deg / latLonResolution
	// Match Stratux behavior: truncate toward zero.
	wk := int32(v)
	u := uint32(wk) & 0x00FFFFFF
	return [3]byte{byte((u >> 16) & 0xFF), byte((u >> 8) & 0xFF), byte(u & 0xFF)}
}

func encodeAltitude12(altFeet int) uint16 {
	// 25 ft resolution with +1000 ft offset.
	// GDL90 spec / Stratux convention:
	//   - range -1000..101350 -> 0x000..0xFFE
	//   - invalid/unavailable -> 0xFFF
	if altFeet < -1000 || altFeet > 101350 {
		return 0x0FFF
	}
	v := (altFeet + 1000) / 25
	return uint16(v) & 0x0FFF
}

func encodeU12(v int) uint16 {
	if v < 0 {
		return 0
	}
	if v > 0xFFF {
		return 0xFFF
	}
	return uint16(v)
}

func encodeTrack8(deg float64) byte {
	if deg < 0 {
		deg = math.Mod(deg, 360) + 360
	}
	deg = math.Mod(deg, 360)
	return byte(math.Floor((deg + trackResolution/2) / trackResolution))
}

func sanitizeCallsign(s string) string {
	if s == "" {
		s = "STRATUX"
	}
	s = strings.ToUpper(s)
	if len(s) > 8 {
		s = s[:8]
	}
	// Replace unsupported chars with space.
	b := []byte(s)
	for i := range b {
		c := b[i]
		ok := (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || c == ' '
		if !ok {
			b[i] = ' '
		}
	}
	for len(b) < 8 {
		b = append(b, ' ')
	}
	return string(b)
}
