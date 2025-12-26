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
	if s == nil {
		return
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.targets[t.ICAO] = target{traffic: t, seenAt: nowUTC.UTC()}
	s.evictIfNeededLocked()
}

func (s *Store) UpsertMany(nowUTC time.Time, targets []gdl90.Traffic) {
	if s == nil {
		return
	}
	if len(targets) == 0 {
		return
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}

	nowUTC = nowUTC.UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range targets {
		// Skip obviously invalid targets.
		if t.LatDeg == 0 && t.LonDeg == 0 {
			continue
		}
		s.targets[t.ICAO] = target{traffic: t, seenAt: nowUTC}
	}
	s.evictIfNeededLocked()
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
