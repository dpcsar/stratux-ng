package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"stratux-ng/internal/config"
)

func TestAircraftGET_ReturnsSynthesizedProfileByDefault(t *testing.T) {
	cfgPath := writeTempConfigFile(t, "gdl90:\n  dest: '127.0.0.1:4000'\nownship:\n  icao: ABC123\n  callsign: N12345\n")
	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/aircraft")
	if err != nil {
		t.Fatalf("GET /api/aircraft error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	var got AircraftSettingsPayload
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Active != "N12345" {
		t.Fatalf("active=%q want N12345", got.Active)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].ICAO != "ABC123" || got.Profiles[0].Callsign != "N12345" {
		t.Fatalf("unexpected profiles: %+v", got.Profiles)
	}
}

func TestAircraftPOST_AppliesAndSavesMultipleProfiles(t *testing.T) {
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

	payload := AircraftSettingsPayload{
		Active: "Arrow",
		Profiles: []AircraftProfilePayload{
			{Name: "Skyhawk", ICAO: "AAA111", Callsign: "N111SH"},
			{Name: "Arrow", ICAO: "bbb222", Callsign: "N222AR"},
		},
	}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var got AircraftSettingsPayload
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Active != "Arrow" {
		t.Fatalf("active=%q want Arrow", got.Active)
	}
	if len(got.Profiles) != 2 || got.Profiles[1].ICAO != "BBB222" {
		t.Fatalf("unexpected profiles: %+v", got.Profiles)
	}

	select {
	case appliedCfg := <-appliedCh:
		if appliedCfg.Ownship.ICAO != "BBB222" || appliedCfg.Ownship.Callsign != "N222AR" {
			t.Fatalf("expected ownship mirrored to active profile, got %+v", appliedCfg.Ownship)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for Apply")
	}

	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	text := string(onDisk)
	if !strings.Contains(text, "BBB222") {
		t.Fatalf("expected saved profile in yaml, got: %s", text)
	}

	// GET should now reflect the active profile mirrored into ownship.
	resp2, err := http.Get(ts.URL + "/api/settings")
	if err != nil {
		t.Fatalf("GET /api/settings error: %v", err)
	}
	defer resp2.Body.Close()
	var settings SettingsPayload
	if err := json.NewDecoder(resp2.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settings.OwnshipICAO != "BBB222" || settings.OwnshipCallsign != "N222AR" {
		t.Fatalf("expected ownship mirrored to active profile, got %+v", settings)
	}
}

func TestAircraftPOST_UnknownActiveRejected(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)
	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	payload := AircraftSettingsPayload{
		Active: "Nope",
		Profiles: []AircraftProfilePayload{
			{Name: "Skyhawk", ICAO: "AAA111", Callsign: "N111SH"},
		},
	}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
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

func TestAircraftPOST_DuplicateProfileNameRejected(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)
	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	payload := AircraftSettingsPayload{
		Active: "Skyhawk",
		Profiles: []AircraftProfilePayload{
			{Name: "Skyhawk", ICAO: "AAA111", Callsign: "N111SH"},
			{Name: "skyhawk", ICAO: "BBB222", Callsign: "N222AR"},
		},
	}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
}

func TestAircraftPOST_EmptyProfilesRejected(t *testing.T) {
	original := "gdl90:\n  dest: '127.0.0.1:4000'\n"
	cfgPath := writeTempConfigFile(t, original)
	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	body := []byte(`{"active": "x", "profiles": []}`)
	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
}

func TestAircraftPOST_UnknownKeyRejected(t *testing.T) {
	cfgPath := writeTempConfigFile(t, "gdl90:\n  dest: '127.0.0.1:4000'\n")
	store := SettingsStore{ConfigPath: cfgPath}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	body := []byte(`{"active": "x", "profiles": [], "extra": true}`)
	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
}

func TestAircraftPOST_ApplyFailureDoesNotSave(t *testing.T) {
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

	payload := AircraftSettingsPayload{
		Active: "Skyhawk",
		Profiles: []AircraftProfilePayload{
			{Name: "Skyhawk", ICAO: "AAA111", Callsign: "N111SH"},
		},
	}
	b, _ := json.Marshal(payload)

	resp, err := http.Post(ts.URL+"/api/aircraft", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/aircraft error: %v", err)
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
