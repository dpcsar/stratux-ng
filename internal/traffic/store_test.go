package traffic

import (
	"testing"
	"time"

	"stratux-ng/internal/gdl90"
)

func TestStoreApplyMetadataUpdatesTail(t *testing.T) {
	store := NewStore(StoreConfig{MaxTargets: 10, TTL: time.Minute})
	icao, _ := gdl90.ParseICAOHex("ABC123")
	base := gdl90.Traffic{ICAO: icao, Tail: ""}
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

func TestStoreApplyMetadataIgnoredForUnknownTarget(t *testing.T) {
	store := NewStore(StoreConfig{})
	icao, _ := gdl90.ParseICAOHex("ABC123")

	store.Apply(time.Now(), TrafficUpdate{
		ICAO: icao,
		Meta: MetadataUpdate{
			ICAO:    icao,
			Tail:    "N00000",
			HasTail: true,
		},
	})

	if got := len(store.Snapshot(time.Now())); got != 0 {
		t.Fatalf("expected no targets, got %d", got)
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
