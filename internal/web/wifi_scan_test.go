package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWiFiScanHandler_ReturnsNetworks(t *testing.T) {
	old := wifiScanFn
	defer func() { wifiScanFn = old }()

	wifiScanFn = func(ctx context.Context, iface string) ([]WiFiScanNetwork, error) {
		if ctx == nil {
			t.Fatalf("expected ctx")
		}
		if iface != "wlan0" {
			t.Fatalf("iface=%q", iface)
		}
		return []WiFiScanNetwork{{SSID: "B", Signal: 10}, {SSID: "A", Signal: 90}}, nil
	}

	ts := httptest.NewServer(WiFiScanHandler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/?iface=wlan0")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	var got struct {
		Networks []WiFiScanNetwork `json:"networks"`
		LastErr  string            `json:"last_error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.LastErr != "" {
		t.Fatalf("last_error=%q", got.LastErr)
	}
	if len(got.Networks) != 2 {
		t.Fatalf("len=%d", len(got.Networks))
	}
	// Sorted by signal desc then ssid.
	if got.Networks[0].SSID != "A" || got.Networks[0].Signal != 90 {
		t.Fatalf("first=%+v", got.Networks[0])
	}
}
