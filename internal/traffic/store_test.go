package traffic

import (
	"testing"
	"time"

	"stratux-ng/internal/gdl90"
)

func TestStoreApplyMetadataUpdatesTail(t *testing.T) {
	store := NewStore(StoreConfig{MaxTargets: 10, TTL: time.Minute})
	icao, _ := gdl90.ParseICAOHex("ABC123")
	base := gdl90.Traffic{ICAO: icao, Tail: "", LatDeg: 45.0, LonDeg: -122.0}
	store.Upsert(time.Now(), base)

	store.Apply(time.Now(), TrafficUpdate{
		ICAO: icao,
		Meta: MetadataUpdate{
			ICAO:    icao,
			Tail:    "N12345",
			HasTail: true,
		},
	})

	snap := store.Snapshot(time.Now())
	if len(snap) != 1 {
		t.Fatalf("expected 1 target, got %d", len(snap))
	}
	if snap[0].Tail != "N12345" {
		t.Fatalf("tail not updated: %q", snap[0].Tail)
	}
}

func TestStoreApplyMetadataOnlyCreatesBearinglessTarget(t *testing.T) {
	store := NewStore(StoreConfig{})
	icao, _ := gdl90.ParseICAOHex("ABC123")
	now := time.Now()

	store.Apply(now, TrafficUpdate{
		ICAO: icao,
		Meta: MetadataUpdate{
			ICAO:    icao,
			Tail:    "N00000",
			HasTail: true,
		},
	})

	if got := len(store.Snapshot(now)); got != 0 {
		t.Fatalf("expected no positional targets, got %d", got)
	}
	detailed := store.SnapshotDetailed(now)
	if len(detailed) != 1 {
		t.Fatalf("expected 1 detailed target, got %d", len(detailed))
	}
	if detailed[0].PositionValid {
		t.Fatalf("expected bearingless target to lack position")
	}
	if detailed[0].Traffic.Tail != "N00000" {
		t.Fatalf("expected tail to persist, got %q", detailed[0].Traffic.Tail)
	}
}

func TestStoreApplyCarriesForwardMissingTail(t *testing.T) {
	store := NewStore(StoreConfig{MaxTargets: 10, TTL: time.Minute})
	icao, _ := gdl90.ParseICAOHex("ABC123")
	base := gdl90.Traffic{ICAO: icao, Tail: "N77777", LatDeg: 1, LonDeg: 2}
	now := time.Now()
	store.Upsert(now, base)

	upd := TrafficUpdate{
		ICAO: icao,
		Traffic: &gdl90.Traffic{
			ICAO:   icao,
			LatDeg: 2,
			LonDeg: 3,
		},
		Meta: MetadataUpdate{ICAO: icao},
	}
	store.Apply(now.Add(time.Second), upd)

	snap := store.Snapshot(time.Now())
	if got := snap[0].Tail; got != "N77777" {
		t.Fatalf("expected tail to persist, got %q", got)
	}
}

func TestStoreSnapshotDetailedIncludesSquawkAndSource(t *testing.T) {
	store := NewStore(StoreConfig{})
	icao, _ := gdl90.ParseICAOHex("00ABCD")
	now := time.Now()

	store.Apply(now, TrafficUpdate{
		ICAO: icao,
		Traffic: &gdl90.Traffic{
			ICAO:   icao,
			LatDeg: 45.0,
			LonDeg: -122.0,
		},
		Meta: MetadataUpdate{
			ICAO:      icao,
			Squawk:    "1200",
			HasSquawk: true,
		},
		Source: Source1090,
	})

	snap := store.SnapshotDetailed(now)
	if len(snap) != 1 {
		t.Fatalf("expected one snapshot, got %d", len(snap))
	}
	if snap[0].Squawk != "1200" {
		t.Fatalf("expected squawk 1200, got %q", snap[0].Squawk)
	}
	if !snap[0].PositionValid {
		t.Fatalf("expected position-valid target")
	}
	if snap[0].Source != Source1090 {
		t.Fatalf("expected 1090 source, got %q", snap[0].Source)
	}
}
