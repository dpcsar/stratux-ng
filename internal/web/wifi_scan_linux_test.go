//go:build linux

package web

import "testing"

func TestSplitNMCLITerseLine_UnescapesColonAndBackslash(t *testing.T) {
	line := "My\\:SSID:70:WPA2"
	parts := splitNMCLITerseLine(line)
	if len(parts) != 3 {
		t.Fatalf("len=%d parts=%v", len(parts), parts)
	}
	if parts[0] != "My:SSID" {
		t.Fatalf("ssid=%q", parts[0])
	}
	if parts[1] != "70" {
		t.Fatalf("signal=%q", parts[1])
	}
	if parts[2] != "WPA2" {
		t.Fatalf("security=%q", parts[2])
	}

	line2 := "Backslash\\\\Net:55:"
	parts2 := splitNMCLITerseLine(line2)
	if len(parts2) != 3 {
		t.Fatalf("len=%d parts=%v", len(parts2), parts2)
	}
	if parts2[0] != "Backslash\\Net" {
		t.Fatalf("ssid=%q", parts2[0])
	}
}
