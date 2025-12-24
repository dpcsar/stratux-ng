//go:build linux

package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDNSMasqLeasesFile_FiltersExpiredAndSorts(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "dnsmasq.leases")

	// dnsmasq format: expiry_epoch mac ip hostname clientid
	// Include one expired and two active leases.
	now := time.Unix(1_700_000_000, 0).UTC()
	content := "" +
		"1699999000 aa:bb:cc:dd:ee:ff 192.168.10.10 oldhost *\n" + // expired
		"1700003600 11:22:33:44:55:66 192.168.10.12 ipad *\n" +
		"1700007200 11:22:33:44:55:77 192.168.10.11 nexus *\n"

	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := parseDNSMasqLeasesFile(p, now)
	if err != nil {
		t.Fatalf("parseDNSMasqLeasesFile: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d got=%v", len(got), got)
	}

	// Sorted by hostname then IP.
	if got[0].Hostname != "ipad" || got[0].IP != "192.168.10.12" {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].Hostname != "nexus" || got[1].IP != "192.168.10.11" {
		t.Fatalf("second=%+v", got[1])
	}

	if got[0].ExpiresUTC == "" || got[1].ExpiresUTC == "" {
		t.Fatalf("expected ExpiresUTC to be set")
	}
}
