package traffic

import (
	"encoding/hex"
	"strconv"
	"strings"
)

const dump978UplinkBytes = 432

// ParseDump978RawUplinkLine parses a dump978/dump978-fa raw uplink line.
//
// Expected format (prefix and trailing fields may vary):
//
//	+<hex>;rs=<n>;ss=<n>;
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

// ParseDump978RawUplinkLineWithMeta parses a dump978/dump978-fa raw uplink line
// and returns the 432-byte payload plus optional signal strength metadata.
//
// Expected format (fields may vary):
//
//	+<hex>;rs=<n>;ss=<n>;
//
// When present, ss is an amplitude value (often 0..1000).
func ParseDump978RawUplinkLineWithMeta(line []byte) (payload []byte, ss int, hasSS bool, ok bool) {
	if len(line) == 0 {
		return nil, 0, false, false
	}

	s := strings.TrimSpace(string(line))
	if s == "" {
		return nil, 0, false, false
	}
	parts := strings.Split(s, ";")
	if len(parts) == 0 {
		return nil, 0, false, false
	}
	first := parts[0]
	if first == "" || first[0] != '+' {
		return nil, 0, false, false
	}
	hexStr := first[1:]
	if hexStr == "" || len(hexStr)%2 != 0 {
		return nil, 0, false, false
	}

	inBytes := len(hexStr) / 2
	if inBytes > dump978UplinkBytes {
		return nil, 0, false, false
	}
	if inBytes < dump978UplinkBytes {
		hexStr = hexStr + strings.Repeat("00", dump978UplinkBytes-inBytes)
		inBytes = dump978UplinkBytes
	}

	out := make([]byte, dump978UplinkBytes)
	if _, err := hex.Decode(out, []byte(hexStr)); err != nil {
		return nil, 0, false, false
	}

	for _, p := range parts[1:] {
		if len(p) >= 3 && p[0] == 's' && p[1] == 's' && p[2] == '=' {
			v, err := strconv.Atoi(strings.TrimSpace(p[3:]))
			if err == nil {
				return out, v, true, true
			}
		}
	}

	return out, 0, false, true
}
