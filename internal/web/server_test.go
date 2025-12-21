package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAPIStatus(t *testing.T) {
	st := NewStatus()
	st.SetStatic("127.0.0.1:4000", "1s", map[string]any{"scenario": false})

	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil))
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
	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil))
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

func TestAPIScenarios_ListsYAMLFiles(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "configs", "scenarios")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "a.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "b.yml"), []byte("x: 2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "ignore.txt"), []byte("nope\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	st := NewStatus()
	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/scenarios")
	if err != nil {
		t.Fatalf("get scenarios: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code=%d", resp.StatusCode)
	}

	var out struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(out.Paths) != 2 {
		t.Fatalf("paths=%v", out.Paths)
	}
	if out.Paths[0] != "./configs/scenarios/a.yaml" {
		t.Fatalf("paths[0]=%q", out.Paths[0])
	}
	if out.Paths[1] != "./configs/scenarios/b.yml" {
		t.Fatalf("paths[1]=%q", out.Paths[1])
	}
}
