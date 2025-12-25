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
	scenarioEnable := false
	scenarioPath := ""
	scenarioStart := ""
	scenarioLoop := false
	trafficEnable := false
	wifiSubnet := "192.168.10.0/24"
	wifiApIP := "192.168.10.1"
	wifiDhcpStart := "192.168.10.50"
	wifiDhcpEnd := "192.168.10.150"
	wifiUplinkEnable := false
	wifiClientNetworks := []WiFiClientNetwork{}
	wifiInetPassthrough := false
	payload := SettingsPayloadIn{
		GDL90Dest:                      &dest,
		Interval:                       &interval,
		WiFiSubnetCIDR:                 &wifiSubnet,
		WiFiAPIp:                       &wifiApIP,
		WiFiDHCPStart:                  &wifiDhcpStart,
		WiFiDHCPEnd:                    &wifiDhcpEnd,
		WiFiUplinkEnable:               &wifiUplinkEnable,
		WiFiClientNetworks:             &wifiClientNetworks,
		WiFiInternetPassThroughEnabled: &wifiInetPassthrough,
		ScenarioEnable:                 &scenarioEnable,
		ScenarioPath:                   &scenarioPath,
		ScenarioStartTimeUTC:           &scenarioStart,
		ScenarioLoop:                   &scenarioLoop,
		TrafficEnable:                  &trafficEnable,
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
		if got.Sim.Traffic.Enable {
			t.Fatalf("expected traffic disabled")
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
	scenarioEnable := false
	scenarioPath := ""
	scenarioStart := ""
	scenarioLoop := false
	trafficEnable := false
	wifiSubnet := "192.168.10.0/24"
	wifiApIP := "192.168.10.1"
	wifiDhcpStart := "192.168.10.50"
	wifiDhcpEnd := "192.168.10.150"
	wifiUplinkEnable := false
	wifiClientNetworks := []WiFiClientNetwork{}
	wifiInetPassthrough := false
	payload := SettingsPayloadIn{
		GDL90Dest:                      &dest,
		Interval:                       &interval,
		WiFiSubnetCIDR:                 &wifiSubnet,
		WiFiAPIp:                       &wifiApIP,
		WiFiDHCPStart:                  &wifiDhcpStart,
		WiFiDHCPEnd:                    &wifiDhcpEnd,
		WiFiUplinkEnable:               &wifiUplinkEnable,
		WiFiClientNetworks:             &wifiClientNetworks,
		WiFiInternetPassThroughEnabled: &wifiInetPassthrough,
		ScenarioEnable:                 &scenarioEnable,
		ScenarioPath:                   &scenarioPath,
		ScenarioStartTimeUTC:           &scenarioStart,
		ScenarioLoop:                   &scenarioLoop,
		TrafficEnable:                  &trafficEnable,
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
	scenarioEnable := false
	scenarioPath := ""
	scenarioStart := ""
	scenarioLoop := false
	trafficEnable := false
	wifiSubnet := "192.168.10.0/24"
	wifiApIP := "192.168.10.1"
	wifiDhcpStart := "192.168.10.50"
	wifiDhcpEnd := "192.168.10.150"
	wifiUplinkEnable := false
	wifiClientNetworks := []WiFiClientNetwork{}
	wifiInetPassthrough := false
	payload := SettingsPayloadIn{
		GDL90Dest:                      &dest,
		Interval:                       nil,
		WiFiSubnetCIDR:                 &wifiSubnet,
		WiFiAPIp:                       &wifiApIP,
		WiFiDHCPStart:                  &wifiDhcpStart,
		WiFiDHCPEnd:                    &wifiDhcpEnd,
		WiFiUplinkEnable:               &wifiUplinkEnable,
		WiFiClientNetworks:             &wifiClientNetworks,
		WiFiInternetPassThroughEnabled: &wifiInetPassthrough,
		ScenarioEnable:                 &scenarioEnable,
		ScenarioPath:                   &scenarioPath,
		ScenarioStartTimeUTC:           &scenarioStart,
		ScenarioLoop:                   &scenarioLoop,
		TrafficEnable:                  &trafficEnable,
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
	"wifi_subnet_cidr": "192.168.10.0/24",
	"wifi_ap_ip": "192.168.10.1",
	"wifi_dhcp_start": "192.168.10.50",
	"wifi_dhcp_end": "192.168.10.150",
	"wifi_uplink_enable": false,
	"wifi_client_networks": [],
	"wifi_internet_passthrough_enable": false,
  "scenario_enable": false,
  "scenario_path": "",
  "scenario_start_time_utc": "",
  "scenario_loop": false,
  "traffic_enable": false
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
