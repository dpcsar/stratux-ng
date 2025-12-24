package traffic

import (
	"encoding/json"
	"testing"

	"stratux-ng/internal/gdl90"
)

func TestParseAircraftJSON_WrapperAircraftArray(t *testing.T) {
	raw := json.RawMessage(`{"aircraft":[{"hex":"ABC123","lat":45.6,"lon":-122.8,"alt_geom":4200,"gs":120,"track":180,"flight":"N12345"}]}`)
	out := ParseAircraftJSON(raw)
	if len(out) != 1 {
		t.Fatalf("expected 1 target, got %d", len(out))
	}
	want, _ := gdl90.ParseICAOHex("ABC123")
	if out[0].ICAO != want {
		t.Fatalf("unexpected ICAO")
	}
}

func TestParseAircraftJSON_SingleAircraftObject(t *testing.T) {
	raw := json.RawMessage(`{"hex":"ABC123","lat":45.6,"lon":-122.8,"alt_baro":4200}`)
	out := ParseAircraftJSON(raw)
	if len(out) != 1 {
		t.Fatalf("expected 1 target, got %d", len(out))
	}
}
