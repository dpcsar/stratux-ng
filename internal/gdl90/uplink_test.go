package gdl90

import (
	"bytes"
	"testing"
)

func TestUATUplinkFrame_RoundTripPayload(t *testing.T) {
	payload := make([]byte, 432)
	for i := 0; i < len(payload); i++ {
		payload[i] = byte(i)
	}

	framed := UATUplinkFrame(payload)
	msg, crcOK, err := Unframe(framed)
	if err != nil {
		t.Fatalf("unframe: %v", err)
	}
	if !crcOK {
		t.Fatalf("crc not ok")
	}
	if len(msg) != 1+3+len(payload) {
		t.Fatalf("unexpected msg len: %d", len(msg))
	}
	if msg[0] != 0x07 {
		t.Fatalf("unexpected msg type: %02x", msg[0])
	}
	if msg[1] != 0 || msg[2] != 0 || msg[3] != 0 {
		t.Fatalf("unexpected time bytes: %02x %02x %02x", msg[1], msg[2], msg[3])
	}
	if !bytes.Equal(msg[4:], payload) {
		t.Fatalf("payload mismatch")
	}
}
