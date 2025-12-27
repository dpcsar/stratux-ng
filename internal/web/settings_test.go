package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stratux-ng/internal/config"
)

func writeTempConfigFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	return p
}

func TestSettingsPOST_AppliesAndSaves(t *testing.T) {
	cfgPath := writeTempConfigFile(t, "gdl90:\n  dest: '127.0.0.1:4000'\n")

	appliedCh := make(chan config.Config, 1)
	store := SettingsStore{
		ConfigPath: cfgPath,
		Apply: func(cfg config.Config) error {
			appliedCh <- cfg
			return nil
		},
	}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	dest := "127.0.0.1:5000"
	interval := "250ms"
	icao := "ABC123"
	callsign := "N12345"
	payload := SettingsPayloadIn{
		GDL90Dest:       &dest,
		Interval:        &interval,
		OwnshipICAO:     &icao,
		OwnshipCallsign: &callsign,
	}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(ts.URL+"/api/settings", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/settings error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	select {
	case got := <-appliedCh:
		if got.GDL90.Dest != "127.0.0.1:5000" {
			t.Fatalf("applied dest=%q", got.GDL90.Dest)
		}
		if got.GDL90.Interval != 250*time.Millisecond {
			t.Fatalf("applied interval=%s", got.GDL90.Interval)
		}
		if got.Ownship.ICAO != strings.ToUpper(icao) {
			t.Fatalf("applied ownship icao=%q", got.Ownship.ICAO)
		}
		if got.Ownship.Callsign != callsign {
			t.Fatalf("applied ownship callsign=%q", got.Ownship.Callsign)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for Apply")
	}

	// Ensure it persisted.
	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	text := string(onDisk)
	if !strings.Contains(text, "127.0.0.1:5000") {
		t.Fatalf("expected saved dest in yaml, got: %s", text)
	}
	if !strings.Contains(text, "250ms") {
		t.Fatalf("expected saved interval in yaml, got: %s", text)
	}
}

func TestSettingsPOST_ApplyFailureDoesNotSave(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)

	store := SettingsStore{
		ConfigPath: cfgPath,
		Apply: func(cfg config.Config) error {
			return errors.New("boom")
		},
	}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	dest := "127.0.0.1:5000"
	interval := "2s"
	icao := "ABC123"
	callsign := "N12345"
	payload := SettingsPayloadIn{
		GDL90Dest:       &dest,
		Interval:        &interval,
		OwnshipICAO:     &icao,
		OwnshipCallsign: &callsign,
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/settings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/settings error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(onDisk) != original {
		t.Fatalf("expected config unchanged; got: %s", string(onDisk))
	}
}

func TestSettingsPOST_MissingIntervalRejected(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)

	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	// Interval is required and all other fields must be present (no partial updates).
	dest := "127.0.0.1:5000"
	icao := "ABC123"
	callsign := "N12345"
	payload := SettingsPayloadIn{
		GDL90Dest:       &dest,
		Interval:        nil,
		OwnshipICAO:     &icao,
		OwnshipCallsign: &callsign,
	}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/settings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/settings error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(onDisk) != original {
		t.Fatalf("expected config unchanged; got: %s", string(onDisk))
	}
}

func TestSettingsPOST_DuplicateKeysRejected(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)

	store := SettingsStore{ConfigPath: cfgPath}
	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	// Duplicate gdl90_dest key should be rejected.
	dup := []byte(`{
		"gdl90_dest": "127.0.0.1:5000",
		"gdl90_dest": "127.0.0.1:6000",
		"interval": "1s",
		"ownship_icao": "ABC123",
		"ownship_callsign": "N12345"
	}`)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/settings", bytes.NewReader(dup))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/settings error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(onDisk) != original {
		t.Fatalf("expected config unchanged; got: %s", string(onDisk))
	}
}
