package web

import (
	"os"
	"path/filepath"
	"testing"
)

const testConfigYAML = "gdl90:\n  dest: '127.0.0.1:4000'\n" +
	"aircraft:\n  profiles:\n" +
	"    - name: Stratux\n      icao: F00000\n      callsign: STRATUX\n"

func writeTempConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	return path
}
