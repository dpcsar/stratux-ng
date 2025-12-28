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
	upd, ok := ParseDump978NDJSON(raw)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if upd.Traffic == nil {
		t.Fatalf("expected traffic payload")
	}
	want, _ := gdl90.ParseICAOHex("ABC123")
	if upd.Traffic.ICAO != want {
		t.Fatalf("unexpected ICAO")
	}
	if upd.Traffic.AltFeet != 4200 {
		t.Fatalf("unexpected altitude")
	}
	if upd.Traffic.GroundKt != 120 {
		t.Fatalf("unexpected groundspeed")
	}
}

func TestParseDump978NDJSON_NoPosition_NoTarget(t *testing.T) {
	raw := json.RawMessage(`{"address":"ABC123"}`)
	if upd, ok := ParseDump978NDJSON(raw); ok {
		t.Fatalf("expected failure, got %+v", upd)
	}
}
