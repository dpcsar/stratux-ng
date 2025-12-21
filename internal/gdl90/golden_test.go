package gdl90

import (
	"testing"
	"time"
)

func TestGolden_Heartbeat_StrutuxPacking(t *testing.T) {
	nowUTC := time.Date(2020, time.January, 1, 1, 2, 3, 0, time.UTC) // 01:02:03
	msg := unframeAndCheckCRC(t, HeartbeatFrameAt(nowUTC, true, false))

	want := []byte{0x00, 0x91, 0x01, 0x8B, 0x0E, 0x00, 0x00}
	if len(msg) != len(want) {
		t.Fatalf("unexpected len: got %d want %d", len(msg), len(want))
	}
	for i := range want {
		if msg[i] != want[i] {
			t.Fatalf("byte[%d] mismatch: got 0x%02X want 0x%02X (msg=% X)", i, msg[i], want[i], msg)
		}
	}
}

func TestGolden_OwnshipReport_MinimalVector(t *testing.T) {
	msg := unframeAndCheckCRC(t, OwnshipReportFrame(Ownship{
		ICAO:        [3]byte{0x01, 0x02, 0x03},
		LatDeg:      45.0,
		LonDeg:      -90.0,
		AltFeet:     0,
		HaveNICNACp: true,
		NIC:         8,
		NACp:        8,
		GroundKt:    100,
		TrackDeg:    90,
		Callsign:    "N12345",
		Emitter:     0x01,
		Emergency:   0,
	}))

	want := []byte{
		0x0A,
		0x00,
		0x01, 0x02, 0x03,
		0x20, 0x00, 0x00, // lat 45 deg
		0xC0, 0x00, 0x00, // lon -90 deg
		0x02, 0x89, // alt=0ft => 0x028 and flags 0x09
		0x88,             // NIC/NACp
		0x06, 0x48, 0x00, // gs=100 (0x064), vvel=unknown (0x800)
		0x40, // track=90deg => 64
		0x01, // emitter
		'N', '1', '2', '3', '4', '5', ' ', ' ',
		0x00, // priority/emergency
	}

	if len(msg) != len(want) {
		t.Fatalf("unexpected len: got %d want %d", len(msg), len(want))
	}
	for i := range want {
		if msg[i] != want[i] {
			t.Fatalf("byte[%d] mismatch: got 0x%02X want 0x%02X (msg=% X)", i, msg[i], want[i], msg)
		}
	}
}

func TestGolden_TrafficReport_MinimalVector(t *testing.T) {
	msg := unframeAndCheckCRC(t, TrafficReportFrame(Traffic{
		AddrType:        0x00,
		ICAO:            [3]byte{0x0A, 0x0B, 0x0C},
		LatDeg:          45.0,
		LonDeg:          -90.0,
		AltFeet:         0,
		NIC:             8,
		NACp:            7,
		GroundKt:        120,
		TrackDeg:        90,
		VvelFpm:         0,
		OnGround:        false,
		Extrapolated:    false,
		EmitterCategory: 0x01,
		Tail:            "TGT0001",
		PriorityStatus:  0,
	}))

	want := []byte{
		0x14,
		0x00,
		0x0A, 0x0B, 0x0C,
		0x20, 0x00, 0x00, // lat 45 deg
		0xC0, 0x00, 0x00, // lon -90 deg
		0x02, 0x89, // alt=0ft => 0x028 and indicator bits (track-valid + airborne)
		0x87,
		0x07, 0x80, 0x00, // spd=120 (0x078), vvel=0
		0x40, // track
		0x01, // emitter
		'T', 'G', 'T', '0', '0', '0', '1', ' ',
		0x00,
	}

	if len(msg) != len(want) {
		t.Fatalf("unexpected len: got %d want %d", len(msg), len(want))
	}
	for i := range want {
		if msg[i] != want[i] {
			t.Fatalf("byte[%d] mismatch: got 0x%02X want 0x%02X (msg=% X)", i, msg[i], want[i], msg)
		}
	}
}

func TestGolden_AHRSGDL90LE_LevelFlightVector(t *testing.T) {
	msg := unframeAndCheckCRC(t, AHRSGDL90LEFrame(Attitude{
		Valid:                true,
		RollDeg:              0,
		PitchDeg:             0,
		HeadingDeg:           90,
		SlipSkidDeg:          0,
		YawRateDps:           0,
		GLoad:                1.0,
		IndicatedAirspeedKt:  100,
		PressureAltitudeFeet: 0,
		PressureAltValid:     true,
		VerticalSpeedFpm:     0,
		VerticalSpeedValid:   true,
	}))

	want := []byte{
		0x4C, 0x45, 0x01, 0x01,
		0x00, 0x00, // roll
		0x00, 0x00, // pitch
		0x03, 0x84, // heading 90.0 => 900
		0x00, 0x00, // slip/skid
		0x00, 0x00, // yaw rate
		0x00, 0x0A, // g-load 1.0 => 10
		0x00, 0x64, // airspeed 100
		0x13, 0x88, // palt 0 + 5000.5 => 5000
		0x00, 0x00, // vs
		0x7F, 0xFF,
	}

	if len(msg) != len(want) {
		t.Fatalf("unexpected len: got %d want %d", len(msg), len(want))
	}
	for i := range want {
		if msg[i] != want[i] {
			t.Fatalf("byte[%d] mismatch: got 0x%02X want 0x%02X (msg=% X)", i, msg[i], want[i], msg)
		}
	}
}
