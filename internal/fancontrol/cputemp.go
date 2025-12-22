package fancontrol

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const cpuTempPath = "/sys/class/thermal/thermal_zone0/temp"

func parseCPUTempC(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("cpu temp empty")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse cpu temp %q: %w", s, err)
	}
	if n > 1000 {
		return float64(n) / 1000.0, nil
	}
	return float64(n), nil
}

func readCPUTempCFromPath(path string) (float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read cpu temp: %w", err)
	}
	return parseCPUTempC(string(b))
}

// ReadCPUTempC reads the Raspberry Pi CPU temperature in degrees Celsius.
//
// Linux typically exposes this as a milli-deg-C integer (e.g., 52345) but some
// systems may return an integer already in degrees.
func ReadCPUTempC() (float64, error) {
	return readCPUTempCFromPath(cpuTempPath)
}
