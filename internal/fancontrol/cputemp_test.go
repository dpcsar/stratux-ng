package fancontrol

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCPUTempC_MilliDeg(t *testing.T) {
	v, err := parseCPUTempC("52345\n")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if v < 52.3 || v > 52.4 {
		t.Fatalf("v=%v want ~52.345", v)
	}
}

func TestParseCPUTempC_Degrees(t *testing.T) {
	v, err := parseCPUTempC("52")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if v != 52 {
		t.Fatalf("v=%v want 52", v)
	}
}

func TestParseCPUTempC_Empty(t *testing.T) {
	_, err := parseCPUTempC("\n")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadCPUTempCFromPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "temp")
	if err := os.WriteFile(p, []byte("42000\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	v, err := readCPUTempCFromPath(p)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if v != 42.0 {
		t.Fatalf("v=%v want 42.0", v)
	}
}
