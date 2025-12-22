package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeAHRSPersist struct{}

func (f fakeAHRSPersist) SetLevel() error                         { return nil }
func (f fakeAHRSPersist) ZeroDrift(ctx context.Context) error     { return nil }
func (f fakeAHRSPersist) OrientForward(ctx context.Context) error { return nil }
func (f fakeAHRSPersist) OrientDone(ctx context.Context) error    { return nil }
func (f fakeAHRSPersist) Orientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	return 1, [3]float64{0, 0, 1}, true
}

func TestAPI_AHRSOrientDone_PersistsOrientation(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("gdl90:\n  dest: '127.0.0.1:4000'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	status := NewStatus()
	settings := SettingsStore{ConfigPath: cfgPath}
	h := Handler(status, settings, nil, fakeAHRSPersist{})

	req := httptest.NewRequest(http.MethodPost, "/api/ahrs/orient/done", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}

	// Give filesystem a tiny moment on slower CI, though rename is atomic.
	time.Sleep(10 * time.Millisecond)
	onDisk, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(onDisk)
	if !strings.Contains(text, "orientation") {
		t.Fatalf("expected orientation in saved yaml, got: %s", text)
	}
	if !strings.Contains(text, "forward_axis") {
		t.Fatalf("expected forward_axis in saved yaml, got: %s", text)
	}
}
func TestAPIStatus(t *testing.T) {
	st := NewStatus()
	st.SetStatic("127.0.0.1:4000", "1s", map[string]any{"scenario": false})

	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil, nil))
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
	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil, nil))
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
	ts := httptest.NewServer(Handler(st, SettingsStore{}, nil, nil))
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
