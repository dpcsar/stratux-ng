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
	traffic     gdl90.Traffic
	seenAt      time.Time
	hasPosition bool
	squawk      string
	source      Source
}

type evictionCandidate struct {
	ico  [3]byte
	seen time.Time
}

// TargetSnapshot exposes the latest state for a traffic target, including
// metadata-only entries (no position).
type TargetSnapshot struct {
	Traffic       gdl90.Traffic
	PositionValid bool
	SeenAt        time.Time
	Squawk        string
	Source        Source
}

func hasValidPosition(t gdl90.Traffic) bool {
	return !(t.LatDeg == 0 && t.LonDeg == 0)
}

func icaoLess(a, b [3]byte) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[2] < b[2]
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
	if !exists && upd.Traffic == nil && upd.Meta.Empty() {
		return
	}
	hadPrevious := exists
	if !exists {
		tgt = target{traffic: gdl90.Traffic{ICAO: upd.ICAO}}
		exists = true
	}
	prevTraffic := tgt.traffic
	updated := false

	if upd.Traffic != nil {
		traffic := *upd.Traffic
		if hadPrevious {
			s.carryForwardMetadata(&traffic, prevTraffic, upd.Meta)
		}
		tgt.traffic = traffic
		tgt.seenAt = nowUTC
		tgt.hasPosition = hasValidPosition(traffic)
		updated = true
	}

	if !upd.Meta.Empty() {
		tgt.traffic = applyMetadata(tgt.traffic, upd.Meta)
		tgt.seenAt = nowUTC
		updated = true
	}

	if upd.Meta.HasSquawk {
		tgt.squawk = upd.Meta.Squawk
	}
	if upd.Source != SourceUnknown {
		tgt.source = upd.Source
	}

	if updated {
		s.targets[upd.ICAO] = tgt
	}

	if upd.Traffic != nil {
		s.evictIfNeededLocked()
	}
}

func (s *Store) Snapshot(nowUTC time.Time) []gdl90.Traffic {
	cloned := s.snapshotTargets(nowUTC)
	if len(cloned) == 0 {
		return nil
	}
	out := make([]gdl90.Traffic, 0, len(cloned))
	for _, v := range cloned {
		if !v.hasPosition {
			continue
		}
		out = append(out, v.traffic)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return icaoLess(out[i].ICAO, out[j].ICAO)
	})
	return out
}

// SnapshotDetailed returns all tracked targets, including metadata-only entries
// that lack a valid position.
func (s *Store) SnapshotDetailed(nowUTC time.Time) []TargetSnapshot {
	cloned := s.snapshotTargets(nowUTC)
	if len(cloned) == 0 {
		return nil
	}
	out := make([]TargetSnapshot, 0, len(cloned))
	for _, v := range cloned {
		out = append(out, TargetSnapshot{
			Traffic:       v.traffic,
			PositionValid: v.hasPosition,
			SeenAt:        v.seenAt,
			Squawk:        v.squawk,
			Source:        v.source,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return icaoLess(out[i].Traffic.ICAO, out[j].Traffic.ICAO)
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

func (s *Store) snapshotTargets(nowUTC time.Time) []target {
	if s == nil {
		return nil
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	} else {
		nowUTC = nowUTC.UTC()
	}

	s.mu.Lock()
	if s.cfg.TTL > 0 {
		cutoff := nowUTC.Add(-s.cfg.TTL)
		for k, v := range s.targets {
			if v.seenAt.Before(cutoff) {
				delete(s.targets, k)
			}
		}
	}
	cloned := make([]target, 0, len(s.targets))
	for _, v := range s.targets {
		cloned = append(cloned, v)
	}
	s.mu.Unlock()
	return cloned
}
