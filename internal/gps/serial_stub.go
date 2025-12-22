//go:build !linux

package gps

import (
	"fmt"
	"os"
)

func openSerial(path string, baud int) (*os.File, error) {
	return nil, fmt.Errorf("gps serial not supported on this platform")
}
