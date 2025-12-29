package web

import (
	"sync"
	"time"
)

// AttitudeBroadcaster fanouts high-rate AHRS snapshots to any listeners (e.g. SSE).
// It keeps the most recent value so new subscribers get an immediate sample.
type AttitudeBroadcaster struct {
	mu        sync.RWMutex
	subs      map[int]chan AttitudeSnapshot
	nextID    int
	last      AttitudeSnapshot
	haveLast  bool
	available bool

	headingMu    sync.RWMutex
	headingDeg   float64
	headingValid bool

	smoothMu        sync.Mutex
	rollSmooth      float64
	pitchSmooth     float64
	haveRollSmooth  bool
	havePitchSmooth bool
}

const attitudeSmoothingAlpha = 0.35

func NewAttitudeBroadcaster() *AttitudeBroadcaster {
	return &AttitudeBroadcaster{
		subs: make(map[int]chan AttitudeSnapshot),
	}
}

func (b *AttitudeBroadcaster) SetAvailable(v bool) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.available = v
	b.mu.Unlock()
}

func (b *AttitudeBroadcaster) Available() bool {
	if b == nil {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.available
}

func (b *AttitudeBroadcaster) Subscribe(buffer int) (int, <-chan AttitudeSnapshot) {
	if b == nil {
		return 0, nil
	}
	if buffer <= 0 {
		buffer = 2
	}
	ch := make(chan AttitudeSnapshot, buffer)
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subs[id] = ch
	last := b.last
	have := b.haveLast
	b.mu.Unlock()
	if have {
		select {
		case ch <- last:
		default:
		}
	}
	return id, ch
}

func (b *AttitudeBroadcaster) Unsubscribe(id int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	ch, ok := b.subs[id]
	if ok {
		delete(b.subs, id)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *AttitudeBroadcaster) SetHeading(heading float64, ok bool) {
	if b == nil {
		return
	}
	b.headingMu.Lock()
	b.headingValid = ok
	b.headingDeg = heading
	b.headingMu.Unlock()
}

func (b *AttitudeBroadcaster) Publish(att AttitudeSnapshot) {
	if b == nil {
		return
	}
	b.headingMu.RLock()
	if att.HeadingDeg == nil && b.headingValid {
		v := b.headingDeg
		att.HeadingDeg = &v
	}
	b.headingMu.RUnlock()
	if att.LastUpdateUTC == "" {
		att.LastUpdateUTC = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if att.RollDeg != nil || att.PitchDeg != nil {
		b.smoothMu.Lock()
		if att.RollDeg != nil {
			att.RollDeg = b.applySmoothing(&b.rollSmooth, &b.haveRollSmooth, *att.RollDeg)
		}
		if att.PitchDeg != nil {
			att.PitchDeg = b.applySmoothing(&b.pitchSmooth, &b.havePitchSmooth, *att.PitchDeg)
		}
		b.smoothMu.Unlock()
	}
	b.mu.RLock()
	subs := make([]chan AttitudeSnapshot, 0, len(b.subs))
	for _, ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- att:
		default:
		}
	}
	b.mu.Lock()
	b.last = att
	b.haveLast = true
	b.mu.Unlock()
}

func (b *AttitudeBroadcaster) applySmoothing(state *float64, have *bool, input float64) *float64 {
	if !*have {
		*state = input
		*have = true
		v := *state
		return &v
	}
	*state = *state + attitudeSmoothingAlpha*(input-*state)
	v := *state
	return &v
}
