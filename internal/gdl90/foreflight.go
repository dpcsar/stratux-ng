package gdl90

import "strings"

// ForeFlightIDFrame builds and frames a ForeFlight "ID" message (0x65, subtype 0).
//
// This mirrors Stratux's makeFFIDMessage layout for broad interoperability.
func ForeFlightIDFrame(shortName string, longName string) []byte {
	msg := make([]byte, 39)
	msg[0] = 0x65
	msg[1] = 0x00 // ID message identifier.
	msg[2] = 0x01 // Message version.

	// Serial number (unknown/invalid).
	for i := 3; i <= 10; i++ {
		msg[i] = 0xFF
	}

	shortName = strings.TrimSpace(shortName)
	if shortName == "" {
		shortName = "Stratux"
	}
	if len(shortName) > 8 {
		shortName = shortName[:8]
	}
	copy(msg[11:], []byte(shortName))

	longName = strings.TrimSpace(longName)
	if longName == "" {
		longName = "Stratux-NG"
	}
	if len(longName) > 16 {
		longName = longName[:16]
	}
	copy(msg[19:], []byte(longName))

	// Capabilities mask: 0x01 indicates MSL altitude for Ownship Geometric report.
	msg[38] = 0x01

	return Frame(msg)
}
