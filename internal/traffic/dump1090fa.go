package traffic

import (
	"encoding/json"

	"stratux-ng/internal/gdl90"
)

// ParseDump1090FAAircraftJSON parses the periodically-updated aircraft.json
// produced by dump1090-fa (via --write-json).
//
// The file format is compatible with the common "aircraft.json" wrapper used by
// dump1090 variants, so we reuse the tolerant aircraft JSON parsing.
func ParseDump1090FAAircraftJSON(raw json.RawMessage) []gdl90.Traffic {
	return ParseAircraftJSON(raw)
}
