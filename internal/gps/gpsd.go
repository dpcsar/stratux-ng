package gps

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

const gpsdDefaultAddr = "127.0.0.1:2947"

// dialGPSD connects to gpsd over TCP.
func dialGPSD(ctx context.Context, addr string) (net.Conn, error) {
	if strings.TrimSpace(addr) == "" {
		addr = gpsdDefaultAddr
	}
	d := &net.Dialer{Timeout: 2 * time.Second}
	if ctx == nil {
		return d.Dial("tcp", addr)
	}
	return d.DialContext(ctx, "tcp", addr)
}

// gpsdWatch enables JSON streaming reports.
func gpsdWatch(conn net.Conn) error {
	// scaled=true yields SI units (m/s, meters) and degrees.
	_, err := conn.Write([]byte("?WATCH={\"enable\":true,\"json\":true,\"scaled\":true}\n"))
	return err
}

type gpsdMsgBase struct {
	Class string `json:"class"`
}

type gpsdTPV struct {
	Class string `json:"class"`
	Mode  *int   `json:"mode"`
	Time  string `json:"time"`

	Lat *float64 `json:"lat"`
	Lon *float64 `json:"lon"`

	Alt     *float64 `json:"alt"`
	AltMSL  *float64 `json:"altMSL"`
	SpeedMS *float64 `json:"speed"`
	Track   *float64 `json:"track"`
	ClimbMS *float64 `json:"climb"`

	// Estimated position errors (meters) when available.
	Epx *float64 `json:"epx"`
	Epy *float64 `json:"epy"`
	Eph *float64 `json:"eph"`
	Epv *float64 `json:"epv"`
}

type gpsdSat struct {
	Used bool `json:"used"`
}

type gpsdSKY struct {
	Class      string      `json:"class"`
	HDOP       *float64    `json:"hdop"`
	Satellites []gpsdSat   `json:"satellites"`
	UUsed      *int        `json:"uSat"` // some gpsd versions
	XUsed      interface{} `json:"used"` // ignore if present
}

type gpsdState struct {
	addr   string
	device string

	latDeg float64
	lonDeg float64
	latOK  bool
	lonOK  bool

	altFeet int
	altOK   bool

	groundKt float64
	gsOK     bool

	trackDeg float64
	trkOK    bool

	mode     int
	modeOK   bool
	satsUsed int
	satsOK   bool
	hdop     float64
	hdopOK   bool

	hAccM     float64
	hAccOK    bool
	vAccM     float64
	vAccOK    bool
	vSpeedFPM int
	vsOK      bool

	lastFix time.Time
	valid   bool

	lastErr string
}

func newGPSDState(addr string) *gpsdState {
	// Keep this user-facing label short; the address is configured separately.
	return &gpsdState{addr: addr, device: "gpsd"}
}

func (s *gpsdState) snapshot() Snapshot {
	out := Snapshot{
		Enabled:  true,
		Valid:    s.valid,
		Device:   s.device,
		Source:   "gpsd",
		GPSDAddr: strings.TrimSpace(s.addr),
	}
	if s.altOK {
		v := s.altFeet
		out.AltFeet = &v
	}
	if s.gsOK {
		v := int(math.Round(s.groundKt))
		out.GroundKt = &v
	}
	if s.trkOK {
		v := s.trackDeg
		out.TrackDeg = &v
	}
	if s.modeOK {
		v := s.mode
		out.FixMode = &v
	}
	if s.satsOK {
		v := s.satsUsed
		out.Satellites = &v
	}
	if s.hdopOK {
		v := s.hdop
		out.HDOP = &v
	}
	if s.hAccOK {
		v := s.hAccM
		out.HorizAccM = &v
	}
	if s.vAccOK {
		v := s.vAccM
		out.VertAccM = &v
	}
	if s.vsOK {
		v := s.vSpeedFPM
		out.VertSpeedFPM = &v
	}
	if !s.lastFix.IsZero() {
		out.LastFixUTC = s.lastFix.UTC().Format(time.RFC3339Nano)
	}
	out.LatDeg = s.latDeg
	out.LonDeg = s.lonDeg
	out.LastError = s.lastErr
	return out
}

func (s *gpsdState) applyLine(nowUTC time.Time, line string) (bool, error) {
	var base gpsdMsgBase
	if err := json.Unmarshal([]byte(line), &base); err != nil {
		return false, fmt.Errorf("gpsd json parse failed: %v", err)
	}

	switch strings.ToUpper(strings.TrimSpace(base.Class)) {
	case "TPV":
		var tpv gpsdTPV
		if err := json.Unmarshal([]byte(line), &tpv); err != nil {
			return false, fmt.Errorf("gpsd tpv parse failed: %v", err)
		}
		return s.applyTPV(nowUTC, tpv), nil
	case "SKY":
		var sky gpsdSKY
		if err := json.Unmarshal([]byte(line), &sky); err != nil {
			return false, fmt.Errorf("gpsd sky parse failed: %v", err)
		}
		return s.applySKY(sky), nil
	default:
		// Ignore other gpsd messages (e.g. VERSION/DEVICES/WATCH).
		return false, nil
	}
}

func (s *gpsdState) applyTPV(nowUTC time.Time, tpv gpsdTPV) bool {
	updated := false

	if tpv.Mode != nil {
		s.mode = *tpv.Mode
		s.modeOK = true
		updated = true
	}

	// Accuracy estimates (meters), if provided.
	if tpv.Eph != nil {
		s.hAccM = *tpv.Eph
		s.hAccOK = true
		updated = true
	} else if tpv.Epx != nil && tpv.Epy != nil {
		s.hAccM = math.Sqrt((*tpv.Epx)*(*tpv.Epx) + (*tpv.Epy)*(*tpv.Epy))
		s.hAccOK = true
		updated = true
	}
	if tpv.Epv != nil {
		s.vAccM = *tpv.Epv
		s.vAccOK = true
		updated = true
	}

	fixTime := nowUTC
	if strings.TrimSpace(tpv.Time) != "" {
		if t, err := time.Parse(time.RFC3339Nano, tpv.Time); err == nil {
			fixTime = t.UTC()
		}
	}

	if tpv.Lat != nil {
		s.latDeg = *tpv.Lat
		s.latOK = true
		updated = true
	}
	if tpv.Lon != nil {
		s.lonDeg = *tpv.Lon
		s.lonOK = true
		updated = true
	}

	if tpv.SpeedMS != nil {
		// gpsd scaled speed is m/s.
		s.groundKt = (*tpv.SpeedMS) * 1.9438444924406
		s.gsOK = true
		updated = true
	}
	if tpv.ClimbMS != nil {
		// climb is m/s; convert to ft/min.
		s.vSpeedFPM = int(math.Round((*tpv.ClimbMS) * 196.8503937007874))
		s.vsOK = true
		updated = true
	}
	if tpv.Track != nil {
		s.trackDeg = *tpv.Track
		s.trkOK = true
		updated = true
	}

	altM := tpv.AltMSL
	if altM == nil {
		altM = tpv.Alt
	}
	if altM != nil {
		s.altFeet = int(math.Round((*altM) * 3.280839895013123))
		s.altOK = true
		updated = true
	}

	// Consider valid when mode indicates a fix and lat/lon are present.
	mode := 0
	if s.modeOK {
		mode = s.mode
	}
	if mode >= 2 && s.latOK && s.lonOK {
		s.valid = true
		s.lastFix = fixTime
		updated = true
	}

	return updated
}

func (s *gpsdState) applySKY(sky gpsdSKY) bool {
	updated := false
	if sky.HDOP != nil {
		s.hdop = *sky.HDOP
		s.hdopOK = true
		updated = true
	}
	if len(sky.Satellites) > 0 {
		used := 0
		for _, sat := range sky.Satellites {
			if sat.Used {
				used++
			}
		}
		s.satsUsed = used
		s.satsOK = true
		updated = true
	}
	return updated
}
