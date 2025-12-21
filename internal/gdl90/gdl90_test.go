package gdl90

import (
	"bytes"
	"math"
	"testing"
)

func decodeLatLon24(b [3]byte) float64 {
	u := uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
	// Sign-extend 24-bit two's complement to int32.
	if u&0x00800000 != 0 {
		u |= 0xFF000000
	}
	v := int32(u)
	return float64(v) * latLonResolution
}

func unframeAndCheckCRC(t *testing.T, frame []byte) (msg []byte) {
	t.Helper()
	if len(frame) < 4 {
		t.Fatalf("frame too short: %d", len(frame))
	}
	if frame[0] != flagByte || frame[len(frame)-1] != flagByte {
		t.Fatalf("missing start/end flags")
	}

	// De-escape.
	raw := make([]byte, 0, len(frame))
	for i := 1; i < len(frame)-1; i++ {
		b := frame[i]
		if b == escapeByte {
			i++
			if i >= len(frame)-1 {
				t.Fatalf("truncated escape at end of frame")
			}
			raw = append(raw, frame[i]^escapeXor)
			continue
		}
		raw = append(raw, b)
	}
	if len(raw) < 3 {
		t.Fatalf("unescaped payload too short: %d", len(raw))
	}

	msg = raw[:len(raw)-2]
	crcGot := uint16(raw[len(raw)-2]) | (uint16(raw[len(raw)-1]) << 8)
	crcWant := crc16(msg)
	if crcGot != crcWant {
		t.Fatalf("crc mismatch: got 0x%04X want 0x%04X", crcGot, crcWant)
	}
	return msg
}

func TestFrame_StartEndFlags(t *testing.T) {
	got := Frame([]byte{0x00, 0x01})
	if len(got) < 2 {
		t.Fatalf("frame too short: %d", len(got))
	}
	if got[0] != flagByte {
		t.Fatalf("missing start flag: 0x%02x", got[0])
	}
	if got[len(got)-1] != flagByte {
		t.Fatalf("missing end flag: 0x%02x", got[len(got)-1])
	}
}

func TestFrame_EscapesControlBytes(t *testing.T) {
	// Force both bytes that must be escaped.
	got := Frame([]byte{0x00, flagByte, escapeByte})
	for i := 1; i < len(got)-1; i++ {
		if got[i] == flagByte {
			t.Fatalf("unescaped flag byte found at %d", i)
		}
	}
}

func TestFrame_RoundTripWithCRC(t *testing.T) {
	message := []byte{0x99, 0x00, flagByte, 0x01, escapeByte, 0x02}
	f := Frame(message)
	got := unframeAndCheckCRC(t, f)
	if !bytes.Equal(got, message) {
		t.Fatalf("round-trip mismatch: got % X want % X", got, message)
	}
}

func TestUnframe_RoundTripAndCRCOk(t *testing.T) {
	message := []byte{0x01, 0x02, 0x03, flagByte, escapeByte}
	f := Frame(message)
	msg, ok, err := Unframe(f)
	if err != nil {
		t.Fatalf("Unframe error: %v", err)
	}
	if !ok {
		t.Fatalf("expected crc ok")
	}
	if !bytes.Equal(msg, message) {
		t.Fatalf("unframe mismatch: got % X want % X", msg, message)
	}
}

func TestHeartbeatFrame_Flags(t *testing.T) {
	msg := unframeAndCheckCRC(t, HeartbeatFrame(true, false))
	if len(msg) != 7 {
		t.Fatalf("unexpected heartbeat length: %d", len(msg))
	}
	if msg[0] != 0x00 {
		t.Fatalf("expected msg id 0x00, got 0x%02X", msg[0])
	}
	if msg[1]&0x91 != 0x91 {
		t.Fatalf("expected flags to include 0x01|0x10|0x80, got 0x%02X", msg[1])
	}
	if msg[1]&0x40 != 0x00 {
		t.Fatalf("did not expect maintenance bit set, got 0x%02X", msg[1])
	}
	if msg[2]&0x01 != 0x01 {
		t.Fatalf("expected UTC OK bit in msg[2] set, got 0x%02X", msg[2])
	}
}

func TestStratuxHeartbeatFrame_BitsAndVersion(t *testing.T) {
	msg := unframeAndCheckCRC(t, StratuxHeartbeatFrame(true, true))
	if len(msg) != 2 {
		t.Fatalf("unexpected stratux heartbeat length: %d", len(msg))
	}
	if msg[0] != 0xCC {
		t.Fatalf("expected msg id 0xCC, got 0x%02X", msg[0])
	}
	if msg[1]&0x03 != 0x03 {
		t.Fatalf("expected ahrs+gps bits set, got 0x%02X", msg[1])
	}
	if (msg[1]>>2)&0x3F != 0x01 {
		t.Fatalf("expected protocol version 1, got %d", (msg[1]>>2)&0x3F)
	}
}

func TestForeFlightIDFrameStartsWith65(t *testing.T) {
	f := ForeFlightIDFrame("Stratux", "Stratux-NG")
	if len(f) < 4 {
		t.Fatalf("frame too short: %d", len(f))
	}
	if f[0] != 0x7E || f[len(f)-1] != 0x7E {
		t.Fatalf("missing flags")
	}
	if f[1] != 0x65 {
		t.Fatalf("expected msg type 0x65 at f[1], got 0x%02X", f[1])
	}
}

func TestTrafficReportFrameStartsWith14(t *testing.T) {
	f := TrafficReportFrame(Traffic{
		AddrType:     0x00,
		ICAO:         [3]byte{0xF1, 0x00, 0x01},
		LatDeg:       45.0,
		LonDeg:       -122.0,
		AltFeet:      4500,
		NIC:          8,
		NACp:         8,
		GroundKt:     120,
		TrackDeg:     90,
		VvelFpm:      0,
		OnGround:     false,
		Extrapolated: false,
		Tail:         "TGT0001",
	})
	if len(f) < 4 {
		t.Fatalf("frame too short: %d", len(f))
	}
	if f[0] != 0x7E || f[len(f)-1] != 0x7E {
		t.Fatalf("missing flags")
	}
	if f[1] != 0x14 {
		t.Fatalf("expected msg type 0x14 at f[1], got 0x%02X", f[1])
	}
}

func TestOwnshipReportFrame_CallsignAndValidityBits(t *testing.T) {
	msg := unframeAndCheckCRC(t, OwnshipReportFrame(Ownship{
		ICAO:     [3]byte{0xAA, 0xBB, 0xCC},
		LatDeg:   45.5,
		LonDeg:   -122.5,
		AltFeet:  4500,
		GroundKt: 100,
		TrackDeg: 270,
		Callsign: "n123$%", // includes invalid chars to sanitize
	}))
	if len(msg) != 28 {
		t.Fatalf("unexpected ownship length: %d", len(msg))
	}
	if msg[0] != 0x0A {
		t.Fatalf("expected msg id 0x0A, got 0x%02X", msg[0])
	}
	if msg[2] != 0xAA || msg[3] != 0xBB || msg[4] != 0xCC {
		t.Fatalf("unexpected ICAO bytes: %02X %02X %02X", msg[2], msg[3], msg[4])
	}
	if msg[12]&0x09 != 0x09 {
		t.Fatalf("expected ownship validity bits set (0x09), got 0x%02X", msg[12])
	}
	if msg[13] != 0x88 {
		t.Fatalf("expected NIC/NACp 0x88, got 0x%02X", msg[13])
	}
	wantCallsign := []byte(sanitizeCallsign("n123$%"))
	if !bytes.Equal(msg[19:27], wantCallsign) {
		t.Fatalf("unexpected callsign bytes: got %q want %q", string(msg[19:27]), string(wantCallsign))
	}
}

func TestEncodeTrack8_TruncatesNearWrap(t *testing.T) {
	// Stratux uses truncation (not rounding). Values near 360 should map to 255,
	// not wrap to 0.
	if got := encodeTrack8(359.9); got != 255 {
		t.Fatalf("encodeTrack8(359.9)=%d want 255", got)
	}
	if got := encodeTrack8(359.999); got != 255 {
		t.Fatalf("encodeTrack8(359.999)=%d want 255", got)
	}
	if got := encodeTrack8(360.0); got != 0 {
		t.Fatalf("encodeTrack8(360.0)=%d want 0", got)
	}
	if got := encodeTrack8(-0.1); got != 255 {
		t.Fatalf("encodeTrack8(-0.1)=%d want 255", got)
	}
}

func TestTrafficReportFrame_IndicatorBitsAndNICNACp(t *testing.T) {
	msg := unframeAndCheckCRC(t, TrafficReportFrame(Traffic{
		AddrType:     0x00,
		ICAO:         [3]byte{0xF1, 0x00, 0x01},
		LatDeg:       45.0,
		LonDeg:       -122.0,
		AltFeet:      4500,
		NIC:          8,
		NACp:         7,
		GroundKt:     120,
		TrackDeg:     90,
		VvelFpm:      0,
		OnGround:     true,
		Extrapolated: true,
		Tail:         "TGT-01",
	}))
	if len(msg) != 28 {
		t.Fatalf("unexpected traffic length: %d", len(msg))
	}
	if msg[0] != 0x14 {
		t.Fatalf("expected msg id 0x14, got 0x%02X", msg[0])
	}
	// track valid + extrapolated set; airborne clear when OnGround=true.
	if msg[12]&0x01 == 0 {
		t.Fatalf("expected track-valid bit set, msg[12]=0x%02X", msg[12])
	}
	if msg[12]&0x04 == 0 {
		t.Fatalf("expected extrapolated bit set, msg[12]=0x%02X", msg[12])
	}
	if msg[12]&0x08 != 0 {
		t.Fatalf("expected airborne bit clear for on-ground target, msg[12]=0x%02X", msg[12])
	}
	if msg[13] != 0x87 {
		t.Fatalf("expected NIC/NACp 0x87, got 0x%02X", msg[13])
	}
}

func TestEncodeAltitude12_Clamps(t *testing.T) {
	if got := encodeAltitude12(-2000); got != 0xFFF {
		t.Fatalf("expected invalid sentinel 0xFFF, got 0x%03X", got)
	}
	if got := encodeAltitude12(200000); got != 0xFFF {
		t.Fatalf("expected invalid sentinel 0xFFF, got 0x%03X", got)
	}
}

func TestOwnshipGeometricAltitudeFrame_EncodesFiveFootMSL(t *testing.T) {
	const altFeet = 12345
	msg := unframeAndCheckCRC(t, OwnshipGeometricAltitudeFrame(altFeet))
	if len(msg) != 5 {
		t.Fatalf("unexpected geo-alt length: %d", len(msg))
	}
	if msg[0] != 0x0B {
		t.Fatalf("expected msg id 0x0B, got 0x%02X", msg[0])
	}
	alt5ft := int16(int16(msg[1])<<8 | int16(msg[2]))
	if int(alt5ft)*5 != altFeet/5*5 {
		t.Fatalf("unexpected geo-alt: got %d ft want %d ft (rounded to 5)", int(alt5ft)*5, altFeet/5*5)
	}
	if msg[3] != 0x00 || msg[4] != 0x0A {
		t.Fatalf("unexpected FOM bytes: %02X %02X", msg[3], msg[4])
	}
}

func TestTrafficReportFrame_VerticalVelocityPacking(t *testing.T) {
	msgUp := unframeAndCheckCRC(t, TrafficReportFrame(Traffic{
		AddrType: 0x00,
		ICAO:     [3]byte{0x01, 0x02, 0x03},
		LatDeg:   45.0,
		LonDeg:   -122.0,
		AltFeet:  4500,
		NIC:      8,
		NACp:     8,
		GroundKt: 100,
		TrackDeg: 90,
		VvelFpm:  640, // 10 * 64 fpm
	}))
	vvelUp := uint16(msgUp[15]&0x0F)<<8 | uint16(msgUp[16])
	if vvelUp != 0x00A {
		t.Fatalf("unexpected vvel(+): got 0x%03X want 0x00A", vvelUp)
	}

	msgDn := unframeAndCheckCRC(t, TrafficReportFrame(Traffic{
		AddrType: 0x00,
		ICAO:     [3]byte{0x01, 0x02, 0x04},
		LatDeg:   45.0,
		LonDeg:   -122.0,
		AltFeet:  4500,
		NIC:      8,
		NACp:     8,
		GroundKt: 100,
		TrackDeg: 90,
		VvelFpm:  -640, // -10 * 64 fpm
	}))
	vvelDn := uint16(msgDn[15]&0x0F)<<8 | uint16(msgDn[16])
	if vvelDn != 0xFF6 {
		t.Fatalf("unexpected vvel(-): got 0x%03X want 0xFF6", vvelDn)
	}
}

func TestEncodeLatLon24_DecodeWithinResolution(t *testing.T) {
	cases := []struct {
		lat float64
		lon float64
	}{
		{lat: 0, lon: 0},
		{lat: 45.0, lon: -122.0},
		{lat: -33.865143, lon: 151.2099},
		{lat: 60.123456, lon: -1.234567},
	}
	for _, c := range cases {
		latEnc := encodeLatLon24(c.lat)
		lonEnc := encodeLatLon24(c.lon)

		latDec := decodeLatLon24(latEnc)
		lonDec := decodeLatLon24(lonEnc)

		if math.Abs(latDec-c.lat) > latLonResolution {
			t.Fatalf("lat decode error too large: got %f want %f (res %f)", latDec, c.lat, latLonResolution)
		}
		if math.Abs(lonDec-c.lon) > latLonResolution {
			t.Fatalf("lon decode error too large: got %f want %f (res %f)", lonDec, c.lon, latLonResolution)
		}
	}
}
