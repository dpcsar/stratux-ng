package web

import (
	"encoding/json"
	"testing"

	"stratux-ng/internal/uat978"
)

func TestDecoderStatusSnapshot_JSONIncludesDecodedWhenSet(t *testing.T) {
	snap := DecoderStatusSnapshot{
		Enabled: true,
		Decoded: &UAT978DecodedSnapshot{
			Towers: []uat978.TowerSnapshot{{Key: "(1,2)", LatDeg: 1, LonDeg: 2, MessagesLastMin: 1, HasSignalStrength: false}},
			Weather: uat978.WeatherSnapshot{
				Products: []uat978.ProductSnapshot{{ProductID: 413, MessagesLastMin: 2, ProductName: "Text (DLAC)"}},
				Text:     []uat978.TextReport{{ReceivedUTC: "2025-01-01T00:00:00Z", TowerLatDeg: 1, TowerLonDeg: 2, Text: "METAR..."}},
			},
		},
	}

	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["decoded"]; !ok {
		t.Fatalf("expected decoded key in JSON")
	}
}
