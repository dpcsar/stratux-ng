package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIStatus(t *testing.T) {
	st := NewStatus()
	st.SetStatic("gdl90", "127.0.0.1:4000", "1s", map[string]any{"scenario": false})

	ts := httptest.NewServer(Handler(st, SettingsStore{}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/status")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code=%d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q", ct)
	}

	var snap StatusSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if snap.Service != "stratux-ng" {
		t.Fatalf("service=%q", snap.Service)
	}
	if snap.GDL90Dest != "127.0.0.1:4000" {
		t.Fatalf("gdl90_dest=%q", snap.GDL90Dest)
	}
}

func TestRootPage(t *testing.T) {
	st := NewStatus()
	ts := httptest.NewServer(Handler(st, SettingsStore{}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("get root: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code=%d", resp.StatusCode)
	}
}
