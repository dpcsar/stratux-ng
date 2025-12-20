package gdl90

import "time"

const (
	flagByte   = 0x7E
	escapeByte = 0x7D
	escapeXor  = 0x20
)

// Frame takes an unframed GDL90 message (message ID + payload bytes), appends
// the GDL90 CRC16, applies byte-stuffing, and wraps with 0x7E flags.
func Frame(message []byte) []byte {
	crc := crc16(message)

	// Append CRC little-endian (low byte first), per common GDL90 implementations.
	withCRC := make([]byte, 0, len(message)+2)
	withCRC = append(withCRC, message...)
	withCRC = append(withCRC, byte(crc&0xFF), byte((crc>>8)&0xFF))

	out := make([]byte, 0, 2+len(withCRC)*2)
	out = append(out, flagByte)
	for _, b := range withCRC {
		if b == flagByte || b == escapeByte {
			out = append(out, escapeByte, b^escapeXor)
			continue
		}
		out = append(out, b)
	}
	out = append(out, flagByte)
	return out
}

// HeartbeatFrame builds and frames a standard GDL90 Heartbeat (0x00).
//
// This is the minimum message many clients expect to see once per second.
func HeartbeatFrame(gpsValid bool, maintenanceRequired bool) []byte {
	msg := make([]byte, 7)
	msg[0] = 0x00

	// Byte 1 flags (per common GDL90 ICD usage):
	// - bit0: UAT initialized
	// - bit4: addr talkback (set)
	// - bit6: maintenance required
	// - bit7: UTC OK (set when GPS valid in this simplified implementation)
	flags := byte(0x01) | byte(0x10)
	if gpsValid {
		flags |= 0x80
	}
	if maintenanceRequired {
		flags |= 0x40
	}
	msg[1] = flags

	nowUTC := time.Now().UTC()
	midnightUTC := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	seconds := uint32(nowUTC.Sub(midnightUTC).Seconds())

	// Time since 0000Z. This encoding matches widely-used Stratux behavior.
	msg[2] = byte(((seconds >> 16) << 7) | 0x01) // UTC OK bit
	msg[3] = byte(seconds & 0xFF)
	msg[4] = byte((seconds & 0xFFFF) >> 8)
	msg[5] = 0x00
	msg[6] = 0x00

	return Frame(msg)
}

// StratuxHeartbeatFrame builds and frames the Stratux heartbeat (0xCC).
// Some apps use this to identify Stratux-like devices.
func StratuxHeartbeatFrame(gpsValid bool, ahrsValid bool) []byte {
	msg := make([]byte, 2)
	msg[0] = 0xCC
	b := byte(0)
	if ahrsValid {
		b |= 0x01
	}
	if gpsValid {
		b |= 0x02
	}
	protocolVers := byte(1)
	b |= protocolVers << 2
	msg[1] = b
	return Frame(msg)
}
