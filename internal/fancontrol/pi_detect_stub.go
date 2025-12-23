//go:build !linux

package fancontrol

func isRaspberryPi5() bool {
	return false
}
