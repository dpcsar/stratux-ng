package web

import (
	"sync/atomic"
	"time"
)

type Status struct {
	startUnixNano int64
	framesSent    uint64
	lastTickNano  int64
	mode          atomic.Value // string
	gdl90Dest     atomic.Value // string
	interval      atomic.Value // string
	simInfo       atomic.Value // map[string]any
	attitude      atomic.Value // AttitudeSnapshot
}

func NewStatus() *Status {
	s := &Status{}
	now := time.Now().UTC()
	atomic.StoreInt64(&s.startUnixNano, now.UnixNano())
	atomic.StoreInt64(&s.lastTickNano, 0)
	s.mode.Store("")
	s.gdl90Dest.Store("")
	s.interval.Store("")
	s.simInfo.Store(map[string]any{})
	s.attitude.Store(AttitudeSnapshot{})
	return s
}

// AttitudeSnapshot is a small, UI-friendly view of AHRS output.
//
// Values are in degrees and may be omitted (null) when unknown.
// This is intended for debugging/verification and is not a flight instrument.
type AttitudeSnapshot struct {
	Valid         bool     `json:"valid"`
	RollDeg       *float64 `json:"roll_deg,omitempty"`
	PitchDeg      *float64 `json:"pitch_deg,omitempty"`
	HeadingDeg    *float64 `json:"heading_deg,omitempty"`
	LastUpdateUTC string   `json:"last_update_utc,omitempty"`
}

func (s *Status) SetAttitude(nowUTC time.Time, att AttitudeSnapshot) {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	att.LastUpdateUTC = nowUTC.UTC().Format(time.RFC3339Nano)
	s.attitude.Store(att)
}

func (s *Status) SetStatic(mode string, gdl90Dest string, interval string, simInfo map[string]any) {
	if mode != "" {
		s.mode.Store(mode)
	}
	if gdl90Dest != "" {
		s.gdl90Dest.Store(gdl90Dest)
	}
	if interval != "" {
		s.interval.Store(interval)
	}
	if simInfo != nil {
		s.simInfo.Store(simInfo)
	}
}

func (s *Status) MarkTick(nowUTC time.Time, framesSentThisTick int) {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	atomic.StoreInt64(&s.lastTickNano, nowUTC.UnixNano())
	if framesSentThisTick > 0 {
		atomic.AddUint64(&s.framesSent, uint64(framesSentThisTick))
	}
}

type StatusSnapshot struct {
	Service         string           `json:"service"`
	NowUTC          string           `json:"now_utc"`
	UptimeSec       int64            `json:"uptime_sec"`
	Mode            string           `json:"mode"`
	GDL90Dest       string           `json:"gdl90_dest"`
	Interval        string           `json:"interval"`
	FramesSentTotal uint64           `json:"frames_sent_total"`
	LastTickUTC     string           `json:"last_tick_utc,omitempty"`
	Sim             map[string]any   `json:"sim"`
	Attitude        AttitudeSnapshot `json:"attitude"`
}

func (s *Status) Snapshot(nowUTC time.Time) StatusSnapshot {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	start := time.Unix(0, atomic.LoadInt64(&s.startUnixNano)).UTC()
	uptime := nowUTC.Sub(start)
	lastTick := atomic.LoadInt64(&s.lastTickNano)

	snap := StatusSnapshot{
		Service:         "stratux-ng",
		NowUTC:          nowUTC.UTC().Format(time.RFC3339Nano),
		UptimeSec:       int64(uptime.Seconds()),
		Mode:            s.mode.Load().(string),
		GDL90Dest:       s.gdl90Dest.Load().(string),
		Interval:        s.interval.Load().(string),
		FramesSentTotal: atomic.LoadUint64(&s.framesSent),
		Sim:             s.simInfo.Load().(map[string]any),
		Attitude:        s.attitude.Load().(AttitudeSnapshot),
	}
	if lastTick != 0 {
		snap.LastTickUTC = time.Unix(0, lastTick).UTC().Format(time.RFC3339Nano)
	}
	return snap
}
