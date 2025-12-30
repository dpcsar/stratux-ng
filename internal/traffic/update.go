package traffic

import "stratux-ng/internal/gdl90"

// TrafficUpdate merges any available GDL90 payload with metadata so callers can
// keep a unified target record even when some fields (tail, ground speed, etc.)
// are missing in the latest position frame.
type TrafficUpdate struct {
	ICAO    [3]byte
	Traffic *gdl90.Traffic
	Meta    MetadataUpdate
	Source  Source
}

// validICAO returns the update's ICAO or false if unavailable.
func (u TrafficUpdate) validICAO() ([3]byte, bool) {
	var zero [3]byte
	if u.ICAO != zero {
		return u.ICAO, true
	}
	if u.Traffic != nil && u.Traffic.ICAO != zero {
		return u.Traffic.ICAO, true
	}
	if u.Meta.ICAO != zero {
		return u.Meta.ICAO, true
	}
	return zero, false
}

// withDefaults ensures ICAO consistency across the update.
func (u TrafficUpdate) withDefaults() (TrafficUpdate, bool) {
	icao, ok := u.validICAO()
	if !ok {
		return TrafficUpdate{}, false
	}
	if u.ICAO != icao {
		u.ICAO = icao
	}
	if u.Traffic != nil {
		u.Traffic.ICAO = icao
	}
	if u.Meta.ICAO != icao {
		u.Meta.ICAO = icao
	}
	return u, true
}

// Empty reports whether the update contains any position or metadata changes.
func (u TrafficUpdate) Empty() bool {
	return u.Traffic == nil && u.Meta.Empty()
}

// NewTrafficUpdateFromTraffic treats all fields present on t as authoritative.
func NewTrafficUpdateFromTraffic(t gdl90.Traffic) TrafficUpdate {
	meta := MetadataUpdate{ICAO: t.ICAO}
	if t.Tail != "" {
		meta.Tail = t.Tail
		meta.HasTail = true
	}
	meta.GroundKt = t.GroundKt
	meta.HasGround = true
	meta.TrackDeg = t.TrackDeg
	meta.HasTrack = true
	meta.VvelFpm = t.VvelFpm
	meta.HasVvel = true
	meta.AltFeet = t.AltFeet
	meta.HasAlt = true
	meta.OnGround = t.OnGround
	meta.HasOnGround = true

	return TrafficUpdate{ICAO: t.ICAO, Traffic: &t, Meta: meta, Source: SourceUnknown}
}
