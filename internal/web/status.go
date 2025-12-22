package web

import (
	"sync/atomic"
	"time"

	"stratux-ng/internal/fancontrol"
	"stratux-ng/internal/gps"
)

type Status struct {
	startUnixNano int64
	framesSent    uint64
	lastTickNano  int64
	gdl90Dest     atomic.Value // string
	interval      atomic.Value // string
	simInfo       atomic.Value // map[string]any
	attitude      atomic.Value // AttitudeSnapshot
	ahrsSensors   atomic.Value // AHRSSensorsSnapshot
	fan           atomic.Value // fancontrol.Snapshot
	gps           atomic.Value // gps.Snapshot
}

func NewStatus() *Status {
	s := &Status{}
	now := time.Now().UTC()
	atomic.StoreInt64(&s.startUnixNano, now.UnixNano())
	atomic.StoreInt64(&s.lastTickNano, 0)
	s.gdl90Dest.Store("")
	s.interval.Store("")
	s.simInfo.Store(map[string]any{})
	s.attitude.Store(AttitudeSnapshot{})
	s.ahrsSensors.Store(AHRSSensorsSnapshot{})
	s.fan.Store(fancontrol.Snapshot{})
	s.gps.Store(gps.Snapshot{Enabled: false})
	return s
}

func (s *Status) SetGPS(_ time.Time, snap gps.Snapshot) {
	// snap is already a UI-friendly struct (strings + optional numbers).
	// Keep this method symmetrical with SetFan/SetAttitude.
	if s == nil {
		return
	}
	s.gps.Store(snap)
}

func (s *Status) SetFan(nowUTC time.Time, snap fancontrol.Snapshot) {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	// Ensure Enabled is reflected even if the service hasn't ticked yet.
	if snap.LastUpdateAt.IsZero() {
		snap.LastUpdateAt = nowUTC.UTC()
	}
	s.fan.Store(snap)
}

// AHRSSensorsSnapshot is a small, UI-friendly view of AHRS hardware health.
// This is intended for debugging/verification and is not a flight instrument.
type AHRSSensorsSnapshot struct {
	Enabled           bool   `json:"enabled"`
	IMUDetected       bool   `json:"imu_detected"`
	BaroDetected      bool   `json:"baro_detected"`
	IMUWorking        bool   `json:"imu_working"`
	BaroWorking       bool   `json:"baro_working"`
	OrientationSet    bool   `json:"orientation_set"`
	ForwardAxis       int    `json:"forward_axis"`
	IMULastUpdateUTC  string `json:"imu_last_update_utc,omitempty"`
	BaroLastUpdateUTC string `json:"baro_last_update_utc,omitempty"`
	LastError         string `json:"last_error,omitempty"`
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
	PressureAltFt *float64 `json:"pressure_alt_ft,omitempty"`
	GLoad         *float64 `json:"g_load,omitempty"`
	GMin          *float64 `json:"g_min,omitempty"`
	GMax          *float64 `json:"g_max,omitempty"`
	LastUpdateUTC string   `json:"last_update_utc,omitempty"`
}

func (s *Status) SetAttitude(nowUTC time.Time, att AttitudeSnapshot) {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	att.LastUpdateUTC = nowUTC.UTC().Format(time.RFC3339Nano)
	s.attitude.Store(att)
}

func (s *Status) SetAHRSSensors(nowUTC time.Time, a AHRSSensorsSnapshot) {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	// Normalize empty timestamps to omit.
	if a.IMULastUpdateUTC == "" && a.IMUWorking {
		a.IMULastUpdateUTC = nowUTC.UTC().Format(time.RFC3339Nano)
	}
	if a.BaroLastUpdateUTC == "" && a.BaroWorking {
		a.BaroLastUpdateUTC = nowUTC.UTC().Format(time.RFC3339Nano)
	}
	s.ahrsSensors.Store(a)
}

func (s *Status) SetStatic(gdl90Dest string, interval string, simInfo map[string]any) {
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
	Service         string              `json:"service"`
	NowUTC          string              `json:"now_utc"`
	UptimeSec       int64               `json:"uptime_sec"`
	GDL90Dest       string              `json:"gdl90_dest"`
	Interval        string              `json:"interval"`
	FramesSentTotal uint64              `json:"frames_sent_total"`
	LastTickUTC     string              `json:"last_tick_utc,omitempty"`
	Sim             map[string]any      `json:"sim"`
	Attitude        AttitudeSnapshot    `json:"attitude"`
	AHRSSensors     AHRSSensorsSnapshot `json:"ahrs"`
	Fan             fancontrol.Snapshot `json:"fan"`
	GPS             gps.Snapshot        `json:"gps"`
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
		GDL90Dest:       s.gdl90Dest.Load().(string),
		Interval:        s.interval.Load().(string),
		FramesSentTotal: atomic.LoadUint64(&s.framesSent),
		Sim:             s.simInfo.Load().(map[string]any),
		Attitude:        s.attitude.Load().(AttitudeSnapshot),
		AHRSSensors:     s.ahrsSensors.Load().(AHRSSensorsSnapshot),
		Fan:             s.fan.Load().(fancontrol.Snapshot),
		GPS:             s.gps.Load().(gps.Snapshot),
	}
	if lastTick != 0 {
		snap.LastTickUTC = time.Unix(0, lastTick).UTC().Format(time.RFC3339Nano)
	}
	return snap
}
