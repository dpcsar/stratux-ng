package gdl90

import "fmt"

// Unframe reverses Frame(): it validates 0x7E flag framing, de-escapes the
// payload, and checks the appended CRC16.
//
// It returns the unframed message bytes (message ID + payload, without CRC),
// whether the CRC check passed, and an error for malformed frames.
func Unframe(frame []byte) (msg []byte, crcOK bool, err error) {
	if len(frame) < 4 {
		return nil, false, fmt.Errorf("frame too short: %d", len(frame))
	}
	if frame[0] != flagByte || frame[len(frame)-1] != flagByte {
		return nil, false, fmt.Errorf("missing start/end flags")
	}

	// De-escape and strip flags.
	raw := make([]byte, 0, len(frame))
	for i := 1; i < len(frame)-1; i++ {
		b := frame[i]
		if b == escapeByte {
			i++
			if i >= len(frame)-1 {
				return nil, false, fmt.Errorf("truncated escape at end of frame")
			}
			raw = append(raw, frame[i]^escapeXor)
			continue
		}
		raw = append(raw, b)
	}
	if len(raw) < 3 {
		return nil, false, fmt.Errorf("unescaped payload too short: %d", len(raw))
	}

	msg = raw[:len(raw)-2]
	crcGot := uint16(raw[len(raw)-2]) | (uint16(raw[len(raw)-1]) << 8)
	crcWant := crc16(msg)
	return msg, crcGot == crcWant, nil
}
