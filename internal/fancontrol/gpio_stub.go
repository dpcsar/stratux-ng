//go:build !linux || (!arm && !arm64)

package fancontrol

import "fmt"

// Stub implementation for non-Linux and/or non-ARM platforms.
func openGPIO(pin int) (pwmDriver, error) {
	return nil, fmt.Errorf("fancontrol: gpio unsupported on this platform")
}

var openGPIOFn = openGPIO
