//go:build !linux || (!arm && !arm64)

package fancontrol

import "fmt"

// Stub implementation for non-Linux and/or non-ARM platforms.
func openPWM(pin int) (pwmDriver, error) {
	return nil, fmt.Errorf("fancontrol: pwm unsupported on this platform")
}
