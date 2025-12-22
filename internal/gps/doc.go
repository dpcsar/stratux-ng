package gps

// Package gps provides a minimal NMEA reader for USB serial GNSS receivers.
//
// It is intentionally small and geared toward Stratux-NG bring-up:
// - Parse RMC for lat/lon/ground speed/track
// - Parse GGA for altitude
// - Provide a snapshot for building GDL90 ownship
