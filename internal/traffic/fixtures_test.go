package traffic

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"stratux-ng/internal/gdl90"
)

func TestFixtures_Dump978NDJSON_ToStoreSnapshot(t *testing.T) {
	path := filepath.Join("testdata", "dump978.ndjson")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	now := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)
	s := NewStore(StoreConfig{MaxTargets: 50, TTL: 10 * time.Minute})

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if upd, ok := ParseDump978NDJSON(json.RawMessage(append([]byte(nil), line...))); ok {
			s.Apply(now, upd)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}

	got := s.Snapshot(now)
	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(got))
	}

	icao1, _ := gdl90.ParseICAOHex("ABC123")
	icao2, _ := gdl90.ParseICAOHex("DEF456")
	seen := map[[3]byte]gdl90.Traffic{}
	for _, tgt := range got {
		seen[tgt.ICAO] = tgt
	}
	if _, ok := seen[icao1]; !ok {
		t.Fatalf("missing ABC123")
	}
	if _, ok := seen[icao2]; !ok {
		t.Fatalf("missing DEF456")
	}
	if seen[icao2].OnGround != true {
		t.Fatalf("expected DEF456 on ground")
	}
}

func TestFixtures_Dump1090NDJSON_ToStoreSnapshot(t *testing.T) {
	path := filepath.Join("testdata", "dump1090.ndjson")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	now := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)
	s := NewStore(StoreConfig{MaxTargets: 50, TTL: 10 * time.Minute})

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if upd, ok := ParseDump1090RawJSON(json.RawMessage(append([]byte(nil), line...))); ok {
			s.Apply(now, upd)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}

	got := s.Snapshot(now)
	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(got))
	}

	icao1, _ := gdl90.ParseICAOHex("ABC123")
	icao2, _ := gdl90.ParseICAOHex("DEF456")
	seen := map[[3]byte]gdl90.Traffic{}
	for _, tgt := range got {
		seen[tgt.ICAO] = tgt
	}
	if _, ok := seen[icao1]; !ok {
		t.Fatalf("missing ABC123")
	}
	if _, ok := seen[icao2]; !ok {
		t.Fatalf("missing DEF456")
	}
	if seen[icao2].GroundKt != 0 {
		t.Fatalf("expected DEF456 ground speed updated to 0, got %d", seen[icao2].GroundKt)
	}
	if seen[icao2].Tail != "B77" {
		t.Fatalf("expected DEF456 tail from metadata, got %q", seen[icao2].Tail)
	}
}

func TestStore_TTLExpiresTargets(t *testing.T) {
	s := NewStore(StoreConfig{MaxTargets: 50, TTL: 1 * time.Second})
	icao, _ := gdl90.ParseICAOHex("ABC123")
	old := time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)
	newer := old.Add(2 * time.Second)

	s.Upsert(old, gdl90.Traffic{ICAO: icao, LatDeg: 45.0, LonDeg: -122.0})
	if got := s.Snapshot(newer); len(got) != 0 {
		t.Fatalf("expected 0 targets after TTL purge, got %d", len(got))
	}
}
