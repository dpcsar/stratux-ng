package gdl90

import "testing"

func TestFrame_StartEndFlags(t *testing.T) {
	got := Frame([]byte{0x00, 0x01})
	if len(got) < 2 {
		t.Fatalf("frame too short: %d", len(got))
	}
	if got[0] != flagByte {
		t.Fatalf("missing start flag: 0x%02x", got[0])
	}
	if got[len(got)-1] != flagByte {
		t.Fatalf("missing end flag: 0x%02x", got[len(got)-1])
	}
}

func TestFrame_EscapesControlBytes(t *testing.T) {
	// Force both bytes that must be escaped.
	got := Frame([]byte{0x00, flagByte, escapeByte})
	for i := 1; i < len(got)-1; i++ {
		if got[i] == flagByte {
			t.Fatalf("unescaped flag byte found at %d", i)
		}
	}
}
