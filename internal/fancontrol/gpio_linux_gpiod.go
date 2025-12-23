//go:build linux && (arm || arm64)

package fancontrol

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/warthog618/go-gpiocdev"
)

// openGPIO returns a pwmDriver-compatible wrapper which drives the given BCM GPIO
// as a digital output using the Linux GPIO character device (libgpiod).
//
// This is intended for 2-wire fans driven by a transistor/MOSFET on a hat.
// It maps any duty > 0 to ON and duty == 0 to OFF.
func openGPIO(pin int) (pwmDriver, error) {
	if pin <= 0 {
		return nil, fmt.Errorf("fancontrol: invalid gpio pin %d", pin)
	}

	// On Pi, line names are commonly "GPIO18", etc.
	lineName := fmt.Sprintf("GPIO%d", pin)

	// Try likely chips first (Pi 5 kernel variants can expose header GPIOs on gpiochip0
	// and sometimes additional chips exist).
	chipCandidates := []string{"/dev/gpiochip0", "/dev/gpiochip4"}
	entries, _ := os.ReadDir("/dev")
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "gpiochip") {
			chipCandidates = append(chipCandidates, filepath.Join("/dev", name))
		}
	}

	for _, chipPath := range chipCandidates {
		chip, err := gpiocdev.NewChip(chipPath)
		if err != nil {
			continue
		}
		offset, err := chip.FindLine(lineName)
		if err != nil {
			_ = chip.Close()
			continue
		}
		line, err := chip.RequestLine(offset, gpiocdev.AsOutput(0), gpiocdev.WithConsumer("stratux-ng-fan"))
		if err != nil {
			_ = chip.Close()
			continue
		}
		return &gpiodGPIO{chip: chip, line: line}, nil
	}

	return nil, fmt.Errorf("fancontrol: gpio line %q not found (or busy)", lineName)
}

var openGPIOFn = openGPIO

type gpiodGPIO struct {
	chip *gpiocdev.Chip
	line *gpiocdev.Line
}

func (g *gpiodGPIO) SetFrequencyHz(hz int) error {
	// Digital on/off backend ignores PWM frequency.
	return nil
}

func (g *gpiodGPIO) SetDutyPercent(p float64) error {
	if g == nil || g.line == nil {
		return fmt.Errorf("fancontrol: gpio driver not initialized")
	}
	v := 0
	if p > 0 {
		v = 1
	}
	return g.line.SetValue(v)
}

func (g *gpiodGPIO) Close() error {
	if g == nil || g.line == nil {
		return nil
	}
	// Graceful shutdown: turn fan OFF.
	_ = g.line.SetValue(0)
	err1 := g.line.Close()
	g.line = nil
	if g.chip != nil {
		_ = g.chip.Close()
		g.chip = nil
	}
	return err1
}
