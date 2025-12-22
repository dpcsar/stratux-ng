package gps

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type nmeaSentence struct {
	Type string
	// Fields is the comma-split NMEA payload (excluding $ and checksum).
	Fields []string
}

func parseNMEASentence(line string) (nmeaSentence, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "$") {
		return nmeaSentence{}, fmt.Errorf("nmea: missing '$'")
	}
	star := strings.LastIndexByte(line, '*')
	if star == -1 {
		return nmeaSentence{}, fmt.Errorf("nmea: missing checksum")
	}
	payload := line[1:star]
	ck := strings.TrimSpace(line[star+1:])
	if len(ck) < 2 {
		return nmeaSentence{}, fmt.Errorf("nmea: short checksum")
	}
	ck = ck[:2]
	want, err := hex.DecodeString(ck)
	if err != nil || len(want) != 1 {
		return nmeaSentence{}, fmt.Errorf("nmea: bad checksum")
	}
	got := byte(0)
	for i := 0; i < len(payload); i++ {
		got ^= payload[i]
	}
	if got != want[0] {
		return nmeaSentence{}, fmt.Errorf("nmea: checksum mismatch")
	}

	parts := strings.Split(payload, ",")
	if len(parts) == 0 {
		return nmeaSentence{}, fmt.Errorf("nmea: empty")
	}
	typeField := parts[0]
	if len(typeField) < 3 {
		return nmeaSentence{}, fmt.Errorf("nmea: short type")
	}
	// Accept GNxxx/GPxxx, etc; normalize to last 3 chars.
	t := typeField
	if len(t) > 3 {
		t = t[len(t)-3:]
	}
	return nmeaSentence{Type: strings.ToUpper(t), Fields: parts}, nil
}

type nmeaState struct {
	device string
	baud   int

	latDeg float64
	lonDeg float64
	latOK  bool
	lonOK  bool

	groundKt float64
	gsOK     bool

	trackDeg float64
	trkOK    bool

	altFeet int
	altOK   bool

	fixQuality   int
	fixQualityOK bool
	satellites   int
	satsOK       bool
	hdop         float64
	hdopOK       bool

	lastFix time.Time
	valid   bool

	lastErr string
}

func (s *nmeaState) apply(nowUTC time.Time, sent nmeaSentence) bool {
	switch sent.Type {
	case "RMC":
		return s.applyRMC(nowUTC, sent.Fields)
	case "GGA":
		return s.applyGGA(nowUTC, sent.Fields)
	default:
		return false
	}
}

func (s *nmeaState) snapshot() Snapshot {
	out := Snapshot{
		Enabled: true,
		Valid:   s.valid,
		Source:  "nmea",
		Device:  s.device,
		Baud:    s.baud,
		LatDeg:  s.latDeg,
		LonDeg:  s.lonDeg,
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
	if s.fixQualityOK {
		v := s.fixQuality
		out.FixQuality = &v
	}
	if s.satsOK {
		v := s.satellites
		out.Satellites = &v
	}
	if s.hdopOK {
		v := s.hdop
		out.HDOP = &v
	}
	if !s.lastFix.IsZero() {
		out.LastFixUTC = s.lastFix.UTC().Format(time.RFC3339Nano)
	}
	out.LastError = s.lastErr
	return out
}

// RMC: Recommended Minimum Specific GNSS Data
// Fields (NMEA 0183 v2.3):
//
//	0: talker+type
//	1: time (hhmmss.sss)
//	2: status (A=active, V=void)
//	3: latitude (ddmm.mmmm)
//	4: N/S
//	5: longitude (dddmm.mmmm)
//	6: E/W
//	7: speed over ground (knots)
//	8: course over ground (deg)
//	9: date (ddmmyy)
func (s *nmeaState) applyRMC(nowUTC time.Time, f []string) bool {
	if len(f) < 10 {
		return false
	}
	status := strings.TrimSpace(f[2])
	if status != "A" {
		// Do not update validity on void fixes.
		return false
	}

	lat, latOK := parseNMEALatLon(f[3], f[4])
	lon, lonOK := parseNMEALatLon(f[5], f[6])
	if latOK {
		s.latDeg = lat
		s.latOK = true
	}
	if lonOK {
		s.lonDeg = lon
		s.lonOK = true
	}

	if gs, ok := parseFloat(f[7]); ok {
		s.groundKt = gs
		s.gsOK = true
	}
	if trk, ok := parseFloat(f[8]); ok {
		s.trackDeg = math.Mod(trk+360.0, 360.0)
		s.trkOK = true
	}

	// If we have lat/lon, treat as a fix.
	if s.latOK && s.lonOK {
		s.lastFix = nowUTC
		s.valid = true
		return true
	}
	return false
}

// GGA: Global Positioning System Fix Data
// Fields:
//
//	0: talker+type
//	1: time
//	2: latitude
//	3: N/S
//	4: longitude
//	5: E/W
//	6: fix quality (0=invalid)
//	7: number of satellites
//	8: HDOP
//	9: altitude (meters)
//
// 10: units (M)
func (s *nmeaState) applyGGA(nowUTC time.Time, f []string) bool {
	if len(f) < 11 {
		return false
	}
	fixQStr := strings.TrimSpace(f[6])
	if fixQStr == "" || fixQStr == "0" {
		return false
	}
	if q, err := strconv.Atoi(fixQStr); err == nil {
		s.fixQuality = q
		s.fixQualityOK = true
	}
	if sats, err := strconv.Atoi(strings.TrimSpace(f[7])); err == nil {
		s.satellites = sats
		s.satsOK = true
	}
	if hdop, ok := parseFloat(f[8]); ok {
		s.hdop = hdop
		s.hdopOK = true
	}

	updated := false
	lat, latOK := parseNMEALatLon(f[2], f[3])
	lon, lonOK := parseNMEALatLon(f[4], f[5])
	if latOK {
		s.latDeg = lat
		s.latOK = true
		updated = true
	}
	if lonOK {
		s.lonDeg = lon
		s.lonOK = true
		updated = true
	}

	altM, ok := parseFloat(f[9])
	if ok {
		s.altFeet = int(math.Round(altM * 3.280839895013123))
		s.altOK = true
		updated = true
	}

	if s.latOK && s.lonOK {
		s.lastFix = nowUTC
		s.valid = true
		return updated
	}
	return false
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseNMEALatLon parses NMEA lat/lon in ddmm.mmmm or dddmm.mmmm plus hemisphere.
//
// For latitude (N/S): ddmm.mmmm
// For longitude (E/W): dddmm.mmmm
func parseNMEALatLon(v string, hemi string) (float64, bool) {
	v = strings.TrimSpace(v)
	hemi = strings.TrimSpace(strings.ToUpper(hemi))
	if v == "" || (hemi != "N" && hemi != "S" && hemi != "E" && hemi != "W") {
		return 0, false
	}

	// Split degrees/minutes at the decimal point by taking the last two digits of the integer part as minutes.
	dot := strings.IndexByte(v, '.')
	intPart := v
	if dot != -1 {
		intPart = v[:dot]
	}
	if len(intPart) < 3 {
		return 0, false
	}

	degPart := intPart[:len(intPart)-2]
	minPart := v[len(intPart)-2:]

	deg, err := strconv.Atoi(degPart)
	if err != nil {
		return 0, false
	}
	mins, err := strconv.ParseFloat(minPart, 64)
	if err != nil {
		return 0, false
	}

	dec := float64(deg) + (mins / 60.0)
	if hemi == "S" || hemi == "W" {
		dec = -dec
	}
	return dec, true
}
