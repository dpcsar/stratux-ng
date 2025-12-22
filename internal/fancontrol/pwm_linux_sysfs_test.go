//go:build linux && (arm || arm64)

package fancontrol

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindPWMChip_AcceptsSymlinkedPWMChip(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "pwm")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Create a real pwmchip directory somewhere else, then symlink it as pwmchip0.
	realChip := filepath.Join(dir, "realchip0")
	if err := os.MkdirAll(realChip, 0o755); err != nil {
		t.Fatalf("MkdirAll realChip: %v", err)
	}
	if err := os.WriteFile(filepath.Join(realChip, "npwm"), []byte("2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile npwm: %v", err)
	}

	link := filepath.Join(base, "pwmchip0")
	if err := os.Symlink(realChip, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	old := pwmSysfsBase
	pwmSysfsBase = base
	t.Cleanup(func() { pwmSysfsBase = old })

	chipPath, channel, err := findPWMChip()
	if err != nil {
		t.Fatalf("findPWMChip: %v", err)
	}
	if chipPath != link {
		t.Fatalf("chipPath=%q want %q", chipPath, link)
	}
	if channel != 0 {
		t.Fatalf("channel=%d want 0", channel)
	}
}
