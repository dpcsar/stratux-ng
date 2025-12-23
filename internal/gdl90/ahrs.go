package gdl90

import "math"

// Attitude represents a minimal attitude / airdata snapshot for Stratux-like
// AHRS messages.
//
// Scaling matches Stratux:
// - roll/pitch/heading: degrees, scaled by 10 (0.1 deg units)
// - slipSkid: degrees, scaled by 10 (0.1 deg units)
// - yawRate: deg/s, scaled by 10 (0.1 deg/s units)
// - gLoad: g, scaled by 1000 (0.001 g units)
// - indicated/true airspeed: knots (integer)
// - pressureAltitudeFeet: feet, encoded as uint16(alt + 5000.5)
// - verticalSpeedFpm: ft/min (integer)
//
// When Valid is false, fields are encoded with Stratux invalid sentinels.
// Note: Stratux will still emit these messages even when invalid.
//
// This is simulator-focused; it does not attempt to model a full AHRS.
//
// References:
// - ForeFlight AHRS: stratux/main/gps.go makeFFAHRSMessage()
// - "LE" AHRS report: stratux/main/gps.go makeAHRSGDL90Report()
//
// (See upstream Stratux for exact packing/behavior.)
//
//nolint:revive // field names match domain terms.
type Attitude struct {
	Valid bool

	RollDeg    float64
	PitchDeg   float64
	HeadingDeg float64

	SlipSkidDeg float64
	YawRateDps  float64
	GLoad       float64

	IndicatedAirspeedKt int
	TrueAirspeedKt      int

	PressureAltitudeFeet float64
	PressureAltValid     bool
	VerticalSpeedFpm     int
	VerticalSpeedValid   bool
}

// ForeFlightAHRSFrame builds and frames the Stratux-style ForeFlight AHRS message
// (0x65, sub-id 0x01).
func ForeFlightAHRSFrame(a Attitude) []byte {
	msg := make([]byte, 12)
	msg[0] = 0x65
	msg[1] = 0x01

	// Values if invalid.
	// Note: Stratux only populates roll/pitch here; heading/airspeeds are left
	// invalid (0xFFFF) even when AHRS is valid.
	pitch := int16(0x7FFF)
	roll := int16(0x7FFF)
	hdg := uint16(0xFFFF)
	ias := uint16(0xFFFF)
	tas := uint16(0xFFFF)

	if a.Valid {
		pitch = deg10(a.PitchDeg)
		roll = deg10(a.RollDeg)
	}

	// Roll.
	msg[2] = byte((roll >> 8) & 0xFF)
	msg[3] = byte(roll & 0xFF)
	// Pitch.
	msg[4] = byte((pitch >> 8) & 0xFF)
	msg[5] = byte(pitch & 0xFF)
	// Heading.
	msg[6] = byte((hdg >> 8) & 0xFF)
	msg[7] = byte(hdg & 0xFF)
	// IAS.
	msg[8] = byte((ias >> 8) & 0xFF)
	msg[9] = byte(ias & 0xFF)
	// TAS.
	msg[10] = byte((tas >> 8) & 0xFF)
	msg[11] = byte(tas & 0xFF)

	return Frame(msg)
}

// AHRSGDL90LEFrame builds and frames the Stratux "LE" AHRS report.
//
// Payload starts with: 0x4C, 0x45, 0x01, 0x01.
func AHRSGDL90LEFrame(a Attitude) []byte {
	msg := make([]byte, 24)
	msg[0] = 0x4C
	msg[1] = 0x45
	msg[2] = 0x01
	msg[3] = 0x01

	// Values if invalid.
	pitch := int16(0x7FFF)
	roll := int16(0x7FFF)
	hdg := int16(0x7FFF)
	slipSkid := int16(0x7FFF)
	yawRate := int16(0x7FFF)
	g := int16(0x7FFF)
	airspeed := int16(0x7FFF)
	palt := uint16(0xFFFF)
	vs := int16(0x7FFF)

	if a.Valid {
		pitch = deg10(a.PitchDeg)
		roll = deg10(a.RollDeg)
		hdg = deg10(a.HeadingDeg)
		slipSkid = deg10(-a.SlipSkidDeg) // Stratux uses negative slip/skid in Levil sentence; keep consistent.
		yawRate = deg10(a.YawRateDps)
		// Stratux uses *10 scaling for g-load in this report (not *1000).
		g = deg10(a.GLoad)
		airspeed = int16(clampI32(int32(a.IndicatedAirspeedKt), 0, 32767))
		if a.PressureAltValid {
			palt = uint16(a.PressureAltitudeFeet + 5000.5)
		}
		if a.VerticalSpeedValid {
			vs = int16(clampI32(int32(a.VerticalSpeedFpm), -32768, 32767))
		}
	}

	// Roll.
	msg[4] = byte((roll >> 8) & 0xFF)
	msg[5] = byte(roll & 0xFF)
	// Pitch.
	msg[6] = byte((pitch >> 8) & 0xFF)
	msg[7] = byte(pitch & 0xFF)
	// Heading.
	msg[8] = byte((hdg >> 8) & 0xFF)
	msg[9] = byte(hdg & 0xFF)
	// Slip/skid.
	msg[10] = byte((slipSkid >> 8) & 0xFF)
	msg[11] = byte(slipSkid & 0xFF)
	// Yaw rate.
	msg[12] = byte((yawRate >> 8) & 0xFF)
	msg[13] = byte(yawRate & 0xFF)
	// G.
	msg[14] = byte((g >> 8) & 0xFF)
	msg[15] = byte(g & 0xFF)
	// IAS.
	msg[16] = byte((airspeed >> 8) & 0xFF)
	msg[17] = byte(airspeed & 0xFF)
	// Pressure altitude.
	msg[18] = byte((palt >> 8) & 0xFF)
	msg[19] = byte(palt & 0xFF)
	// Vertical speed.
	msg[20] = byte((vs >> 8) & 0xFF)
	msg[21] = byte(vs & 0xFF)
	// Reserved.
	msg[22] = 0x7F
	msg[23] = 0xFF

	return Frame(msg)
}

func deg10(deg float64) int16 {
	// Round to nearest, matching Stratux common.RoundToInt16 for these fields.
	v := math.Round(deg * 10)
	return int16(clampI32(int32(v), -32768, 32767))
}

func clampI32(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
