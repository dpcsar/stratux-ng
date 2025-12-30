package traffic

import (
	"encoding/json"
	"testing"

	gdl "stratux-ng/internal/gdl90"
)

func TestParseDump1090RawJSONPosition(t *testing.T) {
	raw := `{
        "Icao_addr": 11256099,
        "DF": 17,
        "CA": 5,
        "TypeCode": 17,
        "SubtypeCode": 0,
        "Position_valid": true,
        "Lat": 41.5,
        "Lng": -88.1,
        "Alt": 4500,
        "NACp": 9,
        "Speed_valid": true,
        "Speed": 150,
        "Track": 123.4,
        "Vvel": 256,
        "OnGround": false,
        "Tail": "N12345",
        "Emitter_category": 3
    }`
	msg, ok := ParseDump1090RawJSON(json.RawMessage(raw))
	if !ok {
		t.Fatalf("expected parse to succeed")
	}
	if msg.Traffic == nil {
		t.Fatalf("expected traffic payload")
	}
	got := *msg.Traffic
	wantICAO, _ := gdl.ParseICAOHex("ABC123")
	if got.ICAO != wantICAO {
		t.Fatalf("unexpected ICAO: %+v", got.ICAO)
	}
	if got.AltFeet != 4500 {
		t.Errorf("alt mismatch: got %d", got.AltFeet)
	}
	if got.GroundKt != 150 {
		t.Errorf("ground speed mismatch: got %d", got.GroundKt)
	}
	if got.TrackDeg != 123.4 {
		t.Errorf("track mismatch: got %.1f", got.TrackDeg)
	}
	if got.NIC == 0 || got.NACp == 0 {
		t.Errorf("expected NIC/NACp to be non-zero, got nic=%d nacp=%d", got.NIC, got.NACp)
	}
	if !msg.Meta.HasTail || msg.Meta.Tail != "N12345" {
		t.Errorf("metadata tail mismatch: %+v", msg.Meta)
	}
}

func TestParseDump1090RawJSONMetadataOnly(t *testing.T) {
	raw := `{
        "Icao_addr": 11256099,
        "Position_valid": false,
        "Tail": "n54321",
        "Speed_valid": true,
        "Speed": 90,
        "Track": 77.0
    }`
	msg, ok := ParseDump1090RawJSON(json.RawMessage(raw))
	if !ok {
		t.Fatalf("expected parse to succeed")
	}
	if msg.Traffic != nil {
		t.Fatalf("did not expect traffic payload")
	}
	if !msg.Meta.HasTail || msg.Meta.Tail != "N54321" {
		t.Fatalf("tail not captured: %+v", msg.Meta)
	}
	if !msg.Meta.HasTrack || msg.Meta.TrackDeg != 77.0 {
		t.Fatalf("track not captured: %+v", msg.Meta)
	}
	if !msg.Meta.HasGround || msg.Meta.GroundKt != 90 {
		t.Fatalf("ground speed not captured: %+v", msg.Meta)
	}
}

func TestParseDump1090RawJSONInvalid(t *testing.T) {
	samples := []string{
		`{"Icao_addr":0}`,
		`{"Icao_addr":1193047,"Position_valid":false}`,
		`{"Icao_addr":1193047,"Position_valid":true}`,
	}
	for idx, raw := range samples {
		if msg, ok := ParseDump1090RawJSON(json.RawMessage(raw)); ok {
			t.Fatalf("sample %d should have failed, got %+v", idx, msg)
		}
	}
}
