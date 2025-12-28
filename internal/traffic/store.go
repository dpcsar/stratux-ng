package traffic

import (
	"sort"
	"sync"
	"time"

	"stratux-ng/internal/gdl90"
)

type StoreConfig struct {
	// MaxTargets limits memory use. When exceeded, oldest targets are evicted.
	MaxTargets int
	// TTL controls how long a target is kept without updates.
	TTL time.Duration
}

type Store struct {
	mu sync.RWMutex

	cfg StoreConfig

	targets map[[3]byte]target
}

type target struct {
	traffic gdl90.Traffic
	seenAt  time.Time
}

type evictionCandidate struct {
	ico  [3]byte
	seen time.Time
}

func (s *Store) evictIfNeededLocked() {
	if s == nil {
		return
	}
	if s.cfg.MaxTargets <= 0 {
		return
	}
	over := len(s.targets) - s.cfg.MaxTargets
	if over <= 0 {
		return
	}

	// Collect and evict oldest in one pass.
	cands := make([]evictionCandidate, 0, len(s.targets))
	for k, v := range s.targets {
		cands = append(cands, evictionCandidate{ico: k, seen: v.seenAt})
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].seen.Before(cands[j].seen)
	})
	if over > len(cands) {
		over = len(cands)
	}
	for i := 0; i < over; i++ {
		delete(s.targets, cands[i].ico)
	}
}

func NewStore(cfg StoreConfig) *Store {
	if cfg.MaxTargets <= 0 {
		cfg.MaxTargets = 200
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 30 * time.Second
	}
	return &Store{
		cfg:     cfg,
		targets: make(map[[3]byte]target),
	}
}

func (s *Store) Upsert(nowUTC time.Time, t gdl90.Traffic) {
	s.Apply(nowUTC, NewTrafficUpdateFromTraffic(t))
}

func (s *Store) UpsertMany(nowUTC time.Time, targets []gdl90.Traffic) {
	if s == nil {
		return
	}
	if len(targets) == 0 {
		return
	}
	for _, t := range targets {
		if t.LatDeg == 0 && t.LonDeg == 0 {
			continue
		}
		s.Apply(nowUTC, NewTrafficUpdateFromTraffic(t))
	}
}

// Apply merges the provided traffic/metadata update into the store, keeping
// prior field values when the latest update omitted them.
func (s *Store) Apply(nowUTC time.Time, update TrafficUpdate) {
	if s == nil {
		return
	}
	if update.Empty() {
		return
	}
	upd, ok := update.withDefaults()
	if !ok {
		return
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	nowUTC = nowUTC.UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	tgt, exists := s.targets[upd.ICAO]
	prevTraffic := tgt.traffic

	if upd.Traffic != nil {
		traffic := *upd.Traffic
		if exists {
			s.carryForwardMetadata(&traffic, prevTraffic, upd.Meta)
		}
		tgt.traffic = traffic
		tgt.seenAt = nowUTC
		exists = true
	}

	if exists && !upd.Meta.Empty() {
		tgt.traffic = applyMetadata(tgt.traffic, upd.Meta)
	}

	if upd.Traffic != nil || (exists && !upd.Meta.Empty()) {
		s.targets[upd.ICAO] = tgt
	}

	if upd.Traffic != nil {
		s.evictIfNeededLocked()
	}
}

func (s *Store) Snapshot(nowUTC time.Time) []gdl90.Traffic {
	if s == nil {
		return nil
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}

	s.mu.Lock()
	// Purge stale.
	if s.cfg.TTL > 0 {
		cutoff := nowUTC.UTC().Add(-s.cfg.TTL)
		for k, v := range s.targets {
			if v.seenAt.Before(cutoff) {
				delete(s.targets, k)
			}
		}
	}

	out := make([]gdl90.Traffic, 0, len(s.targets))
	for _, v := range s.targets {
		out = append(out, v.traffic)
	}
	s.mu.Unlock()

	sort.Slice(out, func(i, j int) bool {
		ai := out[i].ICAO
		aj := out[j].ICAO
		if ai[0] != aj[0] {
			return ai[0] < aj[0]
		}
		if ai[1] != aj[1] {
			return ai[1] < aj[1]
		}
		return ai[2] < aj[2]
	})

	return out
}

func (s *Store) carryForwardMetadata(dst *gdl90.Traffic, prev gdl90.Traffic, meta MetadataUpdate) {
	if dst == nil {
		return
	}
	if !meta.HasTail {
		dst.Tail = prev.Tail
	}
	if !meta.HasGround {
		dst.GroundKt = prev.GroundKt
	}
	if !meta.HasTrack {
		dst.TrackDeg = prev.TrackDeg
	}
	if !meta.HasVvel {
		dst.VvelFpm = prev.VvelFpm
	}
	if !meta.HasAlt {
		dst.AltFeet = prev.AltFeet
	}
	if !meta.HasOnGround {
		dst.OnGround = prev.OnGround
	}
}

func applyMetadata(t gdl90.Traffic, meta MetadataUpdate) gdl90.Traffic {
	if meta.HasTail {
		t.Tail = meta.Tail
	}
	if meta.HasGround {
		t.GroundKt = meta.GroundKt
	}
	if meta.HasTrack {
		t.TrackDeg = meta.TrackDeg
	}
	if meta.HasVvel {
		t.VvelFpm = meta.VvelFpm
	}
	if meta.HasAlt {
		t.AltFeet = meta.AltFeet
	}
	if meta.HasOnGround {
		t.OnGround = meta.OnGround
	}
	return t
}
