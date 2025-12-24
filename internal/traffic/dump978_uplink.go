package traffic

import (
	"encoding/hex"
	"strings"
)

const dump978UplinkBytes = 432

// ParseDump978RawUplinkLine parses a dump978/dump978-fa raw uplink line.
//
// Expected format (prefix and trailing fields may vary):
//   +<hex>;rs=<n>;ss=<n>;
//
// It returns the 432-byte uplink payload (zero-padded if the input is short).
//
// This is used to relay UAT uplinks (FIS-B weather) to EFBs via GDL90 message 0x07.
func ParseDump978RawUplinkLine(line []byte) ([]byte, bool) {
	if len(line) == 0 {
		return nil, false
	}

	// Keep only the hex part before any ';' metadata.
	s := strings.TrimSpace(string(line))
	if s == "" {
		return nil, false
	}
	parts := strings.SplitN(s, ";", 2)
	first := parts[0]
	if first == "" || first[0] != '+' {
		return nil, false
	}
	hexStr := first[1:]
	if hexStr == "" {
		return nil, false
	}
	if len(hexStr)%2 != 0 {
		return nil, false
	}

	inBytes := len(hexStr) / 2
	if inBytes > dump978UplinkBytes {
		return nil, false
	}
	if inBytes < dump978UplinkBytes {
		hexStr = hexStr + strings.Repeat("00", dump978UplinkBytes-inBytes)
		inBytes = dump978UplinkBytes
	}

	out := make([]byte, dump978UplinkBytes)
	if _, err := hex.Decode(out, []byte(hexStr)); err != nil {
		return nil, false
	}
	return out, true
}
