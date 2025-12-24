package traffic

import (
	"encoding/json"
	"testing"

	"stratux-ng/internal/gdl90"
)

func TestParseDump978NDJSON_BasicPosition(t *testing.T) {
	raw := json.RawMessage(`{
		"address":"ABC123",
		"position":{"lat":45.6,"lon":-122.8},
		"geometric_altitude":4200,
		"ground_speed":120.2,
		"true_track":180.0,
		"callsign":"N12345"
	}`)
	out := ParseDump978NDJSON(raw)
	if len(out) != 1 {
		t.Fatalf("expected 1 target, got %d", len(out))
	}
	want, _ := gdl90.ParseICAOHex("ABC123")
	if out[0].ICAO != want {
		t.Fatalf("unexpected ICAO")
	}
	if out[0].AltFeet != 4200 {
		t.Fatalf("unexpected altitude")
	}
	if out[0].GroundKt != 120 {
		t.Fatalf("unexpected groundspeed")
	}
}

func TestParseDump978NDJSON_NoPosition_NoTarget(t *testing.T) {
	raw := json.RawMessage(`{"address":"ABC123"}`)
	out := ParseDump978NDJSON(raw)
	if len(out) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(out))
	}
}
