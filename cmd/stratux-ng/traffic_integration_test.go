package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/gps"
	"stratux-ng/internal/traffic"
)

func loadNDJSONLines(t *testing.T, relPath string) []json.RawMessage {
	t.Helper()

	f, err := os.Open(relPath)
	if err != nil {
		t.Fatalf("open %s: %v", relPath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var out []json.RawMessage
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Copy because scanner.Bytes() is reused.
		out = append(out, json.RawMessage(append([]byte(nil), line...)))
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", relPath, err)
	}
	return out
}

func countTrafficMessages(frames [][]byte) int {
	n := 0
	for _, frame := range frames {
		msg, _, err := gdl90.Unframe(frame)
		if err != nil || len(msg) == 0 {
			continue
		}
		if msg[0] == 0x14 {
			n++
		}
	}
	return n
}

func newTrafficTestConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Config{
		GDL90: config.GDL90Config{Dest: "127.0.0.1:4000", Interval: 1 * time.Second},
		GPS:   config.GPSConfig{Enable: true, HorizontalAccuracyM: 10},
		Ownship: config.OwnshipConfig{
			ICAO:     "ABCDEF",
			Callsign: "STRATUX",
		},
	}
	if err := config.DefaultAndValidate(&cfg); err != nil {
		t.Fatalf("default config: %v", err)
	}
	return cfg
}

func TestTrafficReplay_Dump1090Fixtures_EmitsGDL90Traffic(t *testing.T) {
	// Use a stable time so tests are deterministic.
	now := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)

	store := traffic.NewStore(traffic.StoreConfig{MaxTargets: 50, TTL: 10 * time.Minute})
	path := filepath.Join("..", "..", "internal", "traffic", "testdata", "dump1090-aircraft.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	store.UpsertMany(now, traffic.ParseDump1090FAAircraftJSON(json.RawMessage(data)))

	cfg := newTrafficTestConfig(t)

	alt := 500
	gs := 80
	trk := 123.0
	fixMode := 3
	fixQuality := 1
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.0,
		LonDeg:     -122.0,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		FixMode:    &fixMode,
		FixQuality: &fixQuality,
		LastFixUTC: now.Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, store.Snapshot(now))
	if got := countTrafficMessages(frames); got < 1 {
		t.Fatalf("expected at least 1 traffic (0x14) message, got %d", got)
	}
}

func TestTrafficReplay_Dump978Fixtures_EmitsGDL90Traffic(t *testing.T) {
	now := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)

	store := traffic.NewStore(traffic.StoreConfig{MaxTargets: 50, TTL: 10 * time.Minute})
	lines := loadNDJSONLines(t, filepath.Join("..", "..", "internal", "traffic", "testdata", "dump978.ndjson"))
	for _, raw := range lines {
		store.UpsertMany(now, traffic.ParseDump978NDJSON(raw))
	}

	cfg := newTrafficTestConfig(t)

	alt := 500
	gs := 80
	trk := 123.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.0,
		LonDeg:     -122.0,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		LastFixUTC: now.Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, store.Snapshot(now))
	if got := countTrafficMessages(frames); got < 1 {
		t.Fatalf("expected at least 1 traffic (0x14) message, got %d", got)
	}
}

func TestTrafficReplay_Dump1090AircraftJSON_EmitsGDL90Traffic(t *testing.T) {
	now := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)

	store := traffic.NewStore(traffic.StoreConfig{MaxTargets: 50, TTL: 10 * time.Minute})
	b, err := os.ReadFile(filepath.Join("..", "..", "internal", "traffic", "testdata", "dump1090-aircraft.json"))
	if err != nil {
		t.Fatalf("read dump1090-aircraft.json: %v", err)
	}
	store.UpsertMany(now, traffic.ParseDump1090FAAircraftJSON(json.RawMessage(b)))

	cfg := newTrafficTestConfig(t)

	alt := 500
	gs := 80
	trk := 123.0
	gpsSnap := gps.Snapshot{
		Enabled:    true,
		Valid:      true,
		LatDeg:     45.0,
		LonDeg:     -122.0,
		AltFeet:    &alt,
		GroundKt:   &gs,
		TrackDeg:   &trk,
		LastFixUTC: now.Format(time.RFC3339Nano),
	}

	frames := buildGDL90FramesWithGPS(cfg, now, false, ahrs.Snapshot{}, true, gpsSnap, &headingFuser{}, store.Snapshot(now))
	if got := countTrafficMessages(frames); got < 1 {
		t.Fatalf("expected at least 1 traffic (0x14) message, got %d", got)
	}
}
