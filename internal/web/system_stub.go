//go:build !linux

package web

import "time"

func snapshotDisk(_ time.Time) *DiskSnapshot {
	return nil
}

func snapshotNetwork(_ time.Time) *NetworkSnapshot {
	return nil
}

func Shutdown() error {
	return nil
}

func Reboot() error {
	return nil
}
