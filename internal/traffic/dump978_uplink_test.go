package traffic

import (
	"bytes"
	"testing"
)

func TestParseDump978RawUplinkLine_ParsesAndPads(t *testing.T) {
	// 4 bytes of data, then metadata.
	line := []byte("+01020304;rs=0;ss=123;\n")
	got, ok := ParseDump978RawUplinkLine(line)
	if !ok {
		t.Fatalf("expected ok")
	}
	if len(got) != dump978UplinkBytes {
		t.Fatalf("expected %d bytes, got %d", dump978UplinkBytes, len(got))
	}
	wantPrefix := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(got[:4], wantPrefix) {
		t.Fatalf("prefix mismatch: %x", got[:4])
	}
	for i := 4; i < len(got); i++ {
		if got[i] != 0 {
			t.Fatalf("expected padding zeros at %d, got %x", i, got[i])
		}
	}
}

func TestParseDump978RawUplinkLine_RejectsNonUplink(t *testing.T) {
	if _, ok := ParseDump978RawUplinkLine([]byte("-0102;\n")); ok {
		t.Fatalf("expected not ok")
	}
}

func TestParseDump978RawUplinkLine_RejectsOversize(t *testing.T) {
	// 433 bytes => 866 hex chars.
	hexLen := (dump978UplinkBytes + 1) * 2
	hex := make([]byte, 0, 1+hexLen)
	hex = append(hex, '+')
	for i := 0; i < hexLen; i++ {
		hex = append(hex, '0')
	}
	if _, ok := ParseDump978RawUplinkLine(hex); ok {
		t.Fatalf("expected not ok")
	}
}

func TestParseDump978RawUplinkLineWithMeta_ParsesSS(t *testing.T) {
	line := []byte("+01020304;rs=0;ss=123;\n")
	payload, ss, hasSS, ok := ParseDump978RawUplinkLineWithMeta(line)
	if !ok {
		t.Fatalf("expected ok")
	}
	if len(payload) != dump978UplinkBytes {
		t.Fatalf("expected %d bytes, got %d", dump978UplinkBytes, len(payload))
	}
	if !hasSS {
		t.Fatalf("expected hasSS")
	}
	if ss != 123 {
		t.Fatalf("expected ss=123, got %d", ss)
	}
}

func TestParseDump978RawUplinkLineWithMeta_MissingSS(t *testing.T) {
	line := []byte("+01020304;rs=0;foo=bar;\n")
	_, _, hasSS, ok := ParseDump978RawUplinkLineWithMeta(line)
	if !ok {
		t.Fatalf("expected ok")
	}
	if hasSS {
		t.Fatalf("expected hasSS=false")
	}
}

func TestParseDump978RawUplinkLineWithMeta_InvalidSS(t *testing.T) {
	line := []byte("+01020304;ss=abc;\n")
	_, _, hasSS, ok := ParseDump978RawUplinkLineWithMeta(line)
	if !ok {
		t.Fatalf("expected ok")
	}
	if hasSS {
		t.Fatalf("expected hasSS=false")
	}
}
