package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cfg.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	return path
}

func requireErrEq(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("error=%q want %q", err.Error(), want)
	}
}

func TestLoad_RequiresDest(t *testing.T) {
	path := writeTempConfig(t, "gdl90: {}\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.dest is required")
}

func TestLoad_DefaultsApplied(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GDL90.Interval != 1*time.Second {
		t.Fatalf("interval=%s want 1s", cfg.GDL90.Interval)
	}
	if cfg.GDL90.Mode != "gdl90" {
		t.Fatalf("mode=%q want %q", cfg.GDL90.Mode, "gdl90")
	}
	if cfg.GDL90.TestPayload == "" {
		t.Fatalf("expected non-empty test payload")
	}

	// Simulator defaults should be populated even if sim is absent.
	if cfg.Sim.Ownship.Period <= 0 || cfg.Sim.Ownship.RadiusNm <= 0 || cfg.Sim.Ownship.GroundKt <= 0 {
		t.Fatalf("expected ownship defaults applied")
	}
	if cfg.Sim.Traffic.Count <= 0 || cfg.Sim.Traffic.RadiusNm <= 0 || cfg.Sim.Traffic.Period <= 0 || cfg.Sim.Traffic.GroundKt <= 0 {
		t.Fatalf("expected traffic defaults applied")
	}
}

func TestLoad_RecordRequiresPath(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  record:\n    enable: true\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.record.path is required when gdl90.record.enable is true")
}

func TestLoad_RecordDisallowedInTestMode(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  mode: test\n  record:\n    enable: true\n    path: './x.log'\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.record cannot be used with gdl90.mode=test")
}

func TestLoad_ReplayRequiresPath(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  replay:\n    enable: true\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.replay.path is required when gdl90.replay.enable is true")
}

func TestLoad_ReplayDisallowedInTestMode(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  mode: test\n  replay:\n    enable: true\n    path: './x.log'\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.replay cannot be used with gdl90.mode=test")
}

func TestLoad_ReplaySpeedDefaultsToOne(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  replay:\n    enable: true\n    path: './x.log'\n    speed: 0\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GDL90.Replay.Speed != 1 {
		t.Fatalf("speed=%v want 1", cfg.GDL90.Replay.Speed)
	}
}

func TestLoad_ReplayNegativeSpeedRejected(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  replay:\n    enable: true\n    path: './x.log'\n    speed: -1\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.replay.speed must be > 0")
}

func TestLoad_RecordAndReplayMutuallyExclusive(t *testing.T) {
	path := writeTempConfig(t, "gdl90:\n  dest: '127.0.0.1:4000'\n  record:\n    enable: true\n    path: './a.log'\n  replay:\n    enable: true\n    path: './b.log'\n")
	_, err := Load(path)
	requireErrEq(t, err, "gdl90.record and gdl90.replay cannot both be enabled")
}
