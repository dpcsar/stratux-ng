package uat978

import (
	"math"
	"strings"
)

const UplinkFrameDataBytes = 432

// DecodedUplink is a lightweight decode of a dump978 uplink frame payload.
//
// This is intentionally minimal: enough to power the Towers + Weather web pages.
// It is not a full DO-282/DO-358 decoder.
type DecodedUplink struct {
	TowerLatDeg float64
	TowerLonDeg float64

	// ProductIDs from FIS-B frames.
	ProductIDs []uint32

	// TextReports are DLAC-decoded strings (product 413) split into lines.
	TextReports []string
}

func DecodeUplinkFrame(frame []byte) (DecodedUplink, bool) {
	if len(frame) < UplinkFrameDataBytes {
		return DecodedUplink{}, false
	}
	frame = frame[:UplinkFrameDataBytes]

	rawLat := (uint32(frame[0]) << 15) | (uint32(frame[1]) << 7) | (uint32(frame[2]) >> 1)
	rawLon := ((uint32(frame[2]) & 0x01) << 23) | (uint32(frame[3]) << 15) | (uint32(frame[4]) << 7) | (uint32(frame[5]) >> 1)
	lat := float64(rawLat) * 360.0 / 16777216.0
	lon := float64(rawLon) * 360.0 / 16777216.0
	if lat > 90 {
		lat -= 180
	}
	if lon > 180 {
		lon -= 360
	}

	appDataValid := (uint32(frame[6]) & 0x20) != 0
	out := DecodedUplink{TowerLatDeg: lat, TowerLonDeg: lon}
	if !appDataValid {
		return out, true
	}

	appData := frame[8:UplinkFrameDataBytes]
	pos := 0
	totalLen := len(appData)
	for pos+2 <= totalLen {
		h0 := appData[pos]
		h1 := appData[pos+1]
		frameLen := int(h0)<<1 | int(h1)>>7
		frameType := uint32(h1) & 0x0f
		if frameLen == 0 {
			break
		}
		if pos+2+frameLen > totalLen {
			break
		}
		payload := appData[pos+2 : pos+2+frameLen]
		pos += 2 + frameLen
		if frameType != 0 {
			continue // not FIS-B
		}
		if len(payload) < 2 {
			continue
		}
		productID := ((uint32(payload[0]) & 0x1f) << 6) | (uint32(payload[1]) >> 2)
		out.ProductIDs = append(out.ProductIDs, productID)

		// For text products we DLAC-decode the FIS-B payload.
		if productID == 413 {
			fisb, ok := fisbData(payload)
			if !ok {
				continue
			}
			msg := dlacDecode(fisb)
			for _, line := range splitDLACLines(msg) {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				out.TextReports = append(out.TextReports, line)
			}
		}
	}

	return out, true
}

func fisbData(payload []byte) ([]byte, bool) {
	// UAT FIS-B APDU time format handling (t_opt).
	if len(payload) < 4 {
		return nil, false
	}
	if len(payload) < 3 {
		return nil, false
	}
	tOpt := ((uint32(payload[1]) & 0x01) << 1) | (uint32(payload[2]) >> 7)
	length := len(payload)
	switch tOpt {
	case 0: // Hours, Minutes.
		if length < 4 {
			return nil, false
		}
		return payload[4:], true
	case 1: // Hours, Minutes, Seconds.
		if length < 5 {
			return nil, false
		}
		return payload[5:], true
	case 2: // Month, Day, Hours, Minutes.
		if length < 5 {
			return nil, false
		}
		return payload[5:], true
	case 3: // Month, Day, Hours, Minutes, Seconds.
		if length < 6 {
			return nil, false
		}
		return payload[6:], true
	default:
		return nil, false
	}
}

// DLAC alphabet used by FIS-B text products.
// NOTE: Keep JSON escaping in mind when editing this string. In Go source we
// want escape sequences (\x03, \x1A, \t, \n, etc), not literal control chars.
var dlacAlphabet = []byte("\x03ABCDEFGHIJKLMNOPQRSTUVWXYZ\x1A\t\x1E\n| !\"#$%&'()*+,-./0123456789:;<=>?")

func dlacDecode(data []byte) string {
	// DLAC is 6-bit packed text.
	step := 0
	tab := false
	var b strings.Builder
	b.Grow(len(data))

	for i := 0; i < len(data); i++ {
		var ch uint32
		switch step {
		case 0:
			ch = uint32(data[i]) >> 2
		case 1:
			if i == 0 {
				continue
			}
			ch = ((uint32(data[i-1]) & 0x03) << 4) | (uint32(data[i]) >> 4)
		case 2:
			if i == 0 {
				continue
			}
			ch = ((uint32(data[i-1]) & 0x0f) << 2) | (uint32(data[i]) >> 6)
			i-- // keep i the same for step 3
		case 3:
			ch = uint32(data[i]) & 0x3f
		}

		if tab {
			for ch > 0 {
				b.WriteByte(' ')
				ch--
			}
			tab = false
		} else if ch == 28 {
			// Tab control.
			tab = true
		} else {
			if ch < uint32(len(dlacAlphabet)) {
				b.WriteByte(dlacAlphabet[ch])
			}
		}

		step = (step + 1) & 3
	}

	return b.String()
}

func splitDLACLines(s string) []string {
	// 0x1E (record separator) or 0x03 (ETX) are used for breaks.
	out := make([]string, 0, 4)
	for {
		i := strings.IndexByte(s, 0x1E)
		if i == -1 {
			i = strings.IndexByte(s, 0x03)
			if i == -1 {
				out = append(out, s)
				break
			}
		}
		out = append(out, s[:i])
		s = s[i+1:]
	}
	return out
}

func SignalStrengthDbFromAmplitude(amplitude int) float64 {
	// dump978 reports "ss" as an amplitude (often 0..1000).
	// Convert to an RSSI-like dB reading matching Stratux's convention.
	if amplitude <= 0 {
		return -999
	}
	if amplitude > 1000 {
		amplitude = 1000
	}
	return 20 * math.Log10(float64(amplitude)/1000.0)
}
