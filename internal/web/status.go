package web

import (
	"sync/atomic"
	"time"

	"stratux-ng/internal/decoder"
	"stratux-ng/internal/fancontrol"
	"stratux-ng/internal/gps"
)

type DiskSnapshot struct {
	RootPath       string `json:"root_path,omitempty"`
	RootTotalBytes uint64 `json:"root_total_bytes,omitempty"`
	RootFreeBytes  uint64 `json:"root_free_bytes,omitempty"`
	RootAvailBytes uint64 `json:"root_avail_bytes,omitempty"`
	LastError      string `json:"last_error,omitempty"`
}

type NetworkSnapshot struct {
	LocalAddrs []string `json:"local_addrs,omitempty"`
	LastError  string   `json:"last_error,omitempty"`
}

type Status struct {
	startUnixNano int64
	framesSent    uint64
	lastTickNano  int64
	gdl90Dest     atomic.Value // string
	interval      atomic.Value // string
	staticInfo    atomic.Value // map[string]any
	attitude      atomic.Value // AttitudeSnapshot
	ahrsSensors   atomic.Value // AHRSSensorsSnapshot
	fan           atomic.Value // fancontrol.Snapshot
	gps           atomic.Value // gps.Snapshot
	traffic       atomic.Value // []TrafficSnapshot
	adsb1090      atomic.Value // DecoderStatusSnapshot
	uat978        atomic.Value // DecoderStatusSnapshot
}

func NewStatus() *Status {
	s := &Status{}
	now := time.Now().UTC()
	atomic.StoreInt64(&s.startUnixNano, now.UnixNano())
	atomic.StoreInt64(&s.lastTickNano, 0)
	s.gdl90Dest.Store("")
	s.interval.Store("")
	s.staticInfo.Store(map[string]any{})
	s.attitude.Store(AttitudeSnapshot{})
	s.ahrsSensors.Store(AHRSSensorsSnapshot{})
	s.fan.Store(fancontrol.Snapshot{})
	s.gps.Store(gps.Snapshot{Enabled: false})
	s.traffic.Store([]TrafficSnapshot{})
	s.adsb1090.Store(DecoderStatusSnapshot{Enabled: false})
	s.uat978.Store(DecoderStatusSnapshot{Enabled: false})
	return s
}

// DecoderStatusSnapshot is a UI-friendly view of an external decoder ingest path.
//
// This is intended for bring-up and debugging.
type DecoderStatusSnapshot struct {
	Enabled      bool   `json:"enabled"`
	SerialTag    string `json:"serial_tag,omitempty"`
	Command      string `json:"command,omitempty"`
	JSONEndpoint string `json:"json_endpoint,omitempty"`
	RawEndpoint  string `json:"raw_endpoint,omitempty"`

	Supervisor decoder.Snapshot        `json:"supervisor"`
	Stream     *decoder.NDJSONSnapshot `json:"stream,omitempty"`
	RawStream  *decoder.LineSnapshot   `json:"raw_stream,omitempty"`
}

func (s *Status) SetADSB1090Decoder(_ time.Time, snap DecoderStatusSnapshot) {
	if s == nil {
		return
	}
	s.adsb1090.Store(snap)
}

func (s *Status) SetUAT978Decoder(_ time.Time, snap DecoderStatusSnapshot) {
	if s == nil {
		return
	}
	s.uat978.Store(snap)
}

func (s *Status) SetGPS(_ time.Time, snap gps.Snapshot) {
	// snap is already a UI-friendly struct (strings + optional numbers).
	// Keep this method symmetrical with SetFan/SetAttitude.
	if s == nil {
		return
	}
	s.gps.Store(snap)
}

// TrafficSnapshot is a small, UI-friendly view of a traffic target.
//
// This is intended for visualizing traffic on the web map and is not a
// certified traffic display.
type TrafficSnapshot struct {
	ICAO         string  `json:"icao"`
	Tail         string  `json:"tail,omitempty"`
	LatDeg       float64 `json:"lat_deg"`
	LonDeg       float64 `json:"lon_deg"`
	AltFeet      int     `json:"alt_feet"`
	GroundKt     int     `json:"ground_kt"`
	TrackDeg     float64 `json:"track_deg"`
	VvelFpm      int     `json:"vvel_fpm"`
	OnGround     bool    `json:"on_ground"`
	Extrapolated bool    `json:"extrapolated"`

	// Derived fields for UI.
	LastSeenUTC string  `json:"last_seen_utc,omitempty"`
	AgeSec      float64 `json:"age_sec,omitempty"`

	// Internal timestamp used to compute LastSeenUTC/AgeSec.
	SeenUnixNano int64 `json:"-"`
}

func (s *Status) SetTraffic(nowUTC time.Time, traffic []TrafficSnapshot) {
	if s == nil {
		return
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	// Normalize: stamp all targets with the same "seen" time.
	seen := nowUTC.UTC().UnixNano()
	out := make([]TrafficSnapshot, 0, len(traffic))
	for _, t := range traffic {
		t.SeenUnixNano = seen
		t.LastSeenUTC = ""
		t.AgeSec = 0
		out = append(out, t)
	}
	s.traffic.Store(out)
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
	if att.LastUpdateUTC == "" {
		att.LastUpdateUTC = nowUTC.UTC().Format(time.RFC3339Nano)
	}
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

func (s *Status) SetStatic(gdl90Dest string, interval string, staticInfo map[string]any) {
	if gdl90Dest != "" {
		s.gdl90Dest.Store(gdl90Dest)
	}
	if interval != "" {
		s.interval.Store(interval)
	}
	if staticInfo != nil {
		s.staticInfo.Store(staticInfo)
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
	Service         string                `json:"service"`
	NowUTC          string                `json:"now_utc"`
	UptimeSec       int64                 `json:"uptime_sec"`
	GDL90Dest       string                `json:"gdl90_dest"`
	Interval        string                `json:"interval"`
	FramesSentTotal uint64                `json:"frames_sent_total"`
	LastTickUTC     string                `json:"last_tick_utc,omitempty"`
	Info            map[string]any        `json:"info"`
	Attitude        AttitudeSnapshot      `json:"attitude"`
	AHRSSensors     AHRSSensorsSnapshot   `json:"ahrs"`
	Fan             fancontrol.Snapshot   `json:"fan"`
	GPS             gps.Snapshot          `json:"gps"`
	Traffic         []TrafficSnapshot     `json:"traffic"`
	ADSB1090        DecoderStatusSnapshot `json:"adsb1090"`
	UAT978          DecoderStatusSnapshot `json:"uat978"`
	Disk            *DiskSnapshot         `json:"disk,omitempty"`
	Network         *NetworkSnapshot      `json:"network,omitempty"`
}

func (s *Status) Snapshot(nowUTC time.Time) StatusSnapshot {
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	start := time.Unix(0, atomic.LoadInt64(&s.startUnixNano)).UTC()
	uptime := nowUTC.Sub(start)
	lastTick := atomic.LoadInt64(&s.lastTickNano)

	// Traffic: compute UI-friendly age/last-seen without mutating the stored slice.
	trafficRaw := s.traffic.Load().([]TrafficSnapshot)
	traffic := make([]TrafficSnapshot, 0, len(trafficRaw))
	for _, t := range trafficRaw {
		if t.SeenUnixNano != 0 {
			seenAt := time.Unix(0, t.SeenUnixNano).UTC()
			t.LastSeenUTC = seenAt.Format(time.RFC3339Nano)
			age := nowUTC.UTC().Sub(seenAt).Seconds()
			if age < 0 {
				age = 0
			}
			t.AgeSec = age
		}
		traffic = append(traffic, t)
	}

	snap := StatusSnapshot{
		Service:         "stratux-ng",
		NowUTC:          nowUTC.UTC().Format(time.RFC3339Nano),
		UptimeSec:       int64(uptime.Seconds()),
		GDL90Dest:       s.gdl90Dest.Load().(string),
		Interval:        s.interval.Load().(string),
		FramesSentTotal: atomic.LoadUint64(&s.framesSent),
		Info:            s.staticInfo.Load().(map[string]any),
		Attitude:        s.attitude.Load().(AttitudeSnapshot),
		AHRSSensors:     s.ahrsSensors.Load().(AHRSSensorsSnapshot),
		Fan:             s.fan.Load().(fancontrol.Snapshot),
		GPS:             s.gps.Load().(gps.Snapshot),
		Traffic:         traffic,
		ADSB1090:        s.adsb1090.Load().(DecoderStatusSnapshot),
		UAT978:          s.uat978.Load().(DecoderStatusSnapshot),
		Disk:            snapshotDisk(nowUTC),
		Network:         snapshotNetwork(nowUTC),
	}
	if lastTick != 0 {
		snap.LastTickUTC = time.Unix(0, lastTick).UTC().Format(time.RFC3339Nano)
	}
	return snap
}
