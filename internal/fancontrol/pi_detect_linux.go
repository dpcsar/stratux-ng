//go:build linux

package fancontrol

import (
	"os"
	"strings"
)

func isRaspberryPi5() bool {
	// Prefer device-tree model.
	// Common paths across Pi distros.
	paths := []string{
		"/sys/firmware/devicetree/base/model",
		"/proc/device-tree/model",
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		model := strings.TrimSpace(string(b))
		model = strings.Trim(model, "\x00")
		if strings.Contains(model, "Raspberry Pi 5") {
			return true
		}
	}
	return false
}
