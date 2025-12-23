//go:build linux && (arm || arm64)

package fancontrol

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// sysfsPWM drives a hardware PWM channel via /sys/class/pwm.
//
// Notes:
// - On Raspberry Pi, you typically need `dtoverlay=pwm-2chan` (or equivalent)
//   so that GPIO18 is exposed as a PWM channel under /sys/class/pwm.
// - This backend is chosen for Pi 3/4/5 compatibility (Pi 5 often breaks
//   memory-mapped GPIO libraries).
//
// This driver keeps the same semantics as the old rpio backend:
// - SetFrequencyHz expects the *base* frequency used upstream (they pass
//   PWMFrequency*100 and then use a fixed range of 100), so we divide by 100
//   to get the actual PWM output frequency.
// - SetDutyPercent expects 0..100.

type sysfsPWM struct {
	chipPath string // /sys/class/pwm/pwmchipN
	pwmPath  string // /sys/class/pwm/pwmchipN/pwmM
	channel  int

	periodNS uint64
	enabled  bool
}

var pwmSysfsBase = "/sys/class/pwm"

func openPWM(pin int) (pwmDriver, error) {
	// We currently only support the upstream default: GPIO18.
	if pin != 18 {
		return nil, fmt.Errorf("fancontrol: sysfs pwm supports only pwm_pin=18 for now")
	}

	chipPath, channel, err := findPWMChip()
	if err != nil {
		return nil, err
	}

	d := &sysfsPWM{
		chipPath: chipPath,
		channel:  channel,
		pwmPath:  filepath.Join(chipPath, fmt.Sprintf("pwm%d", channel)),
	}

	if err := d.ensureExported(); err != nil {
		return nil, err
	}
	// Default to enabled once exported (we will set period/duty shortly after).
	if err := d.writeBool("enable", false); err == nil {
		d.enabled = false
	}
	return d, nil
}

func findPWMChip() (chipPath string, channel int, err error) {
	base := pwmSysfsBase
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", 0, fmt.Errorf("fancontrol: read %s: %w", base, err)
	}

	// Prefer pwmchip0 if present (common on Pi).
	preferred := []string{"pwmchip0", "pwmchip1", "pwmchip2"}
	// Note: in sysfs, pwmchipN entries are commonly symlinks, not directories.
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "pwmchip") {
			seen[name] = true
		}
	}
	candidates := make([]string, 0, len(preferred)+len(entries))
	for _, name := range preferred {
		if seen[name] {
			candidates = append(candidates, name)
		}
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "pwmchip") && !contains(candidates, name) {
			candidates = append(candidates, name)
		}
	}

	for _, name := range candidates {
		chip := filepath.Join(base, name)
		n, rerr := readInt(filepath.Join(chip, "npwm"))
		if rerr != nil {
			continue
		}
		if n <= 0 {
			continue
		}
		// We assume channel 0 maps to GPIO18 when pwm-2chan overlay is enabled.
		return chip, 0, nil
	}

	return "", 0, fmt.Errorf("fancontrol: no sysfs pwmchip found (is pwm overlay enabled?)")
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func (d *sysfsPWM) ensureExported() error {
	if _, err := os.Stat(d.pwmPath); err == nil {
		return nil
	}
	// Export channel.
	exportPath := filepath.Join(d.chipPath, "export")
	if err := writeSysfs(exportPath, strconv.Itoa(d.channel)); err != nil {
		// If already exported by someone else, ignore.
		if _, statErr := os.Stat(d.pwmPath); statErr == nil {
			return nil
		}
		return fmt.Errorf("fancontrol: export pwm: %w", err)
	}

	// Wait briefly for sysfs node to appear.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(d.pwmPath); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(d.pwmPath); err != nil {
		return fmt.Errorf("fancontrol: pwm path not created after export: %w", err)
	}
	return nil
}

func (d *sysfsPWM) Close() error {
	// Best-effort: set full duty before disabling.
	_ = d.SetDutyPercent(100)
	_ = d.writeBool("enable", false)
	d.enabled = false
	return nil
}

func (d *sysfsPWM) SetFrequencyHz(hz int) error {
	if hz <= 0 {
		return fmt.Errorf("fancontrol: invalid frequency %d", hz)
	}
	// Match upstream semantics: base frequency / 100 => output frequency.
	outHz := hz / 100
	if outHz <= 0 {
		outHz = 1
	}
	periodNS := uint64(1_000_000_000 / outHz)
	if periodNS == 0 {
		periodNS = 1
	}

	// Disable before changing period/duty (common sysfs requirement).
	_ = d.writeBool("enable", false)
	d.enabled = false

	if err := d.writeUint("period", periodNS); err != nil {
		return err
	}
	d.periodNS = periodNS

	// Re-enable after setting period.
	if err := d.writeBool("enable", true); err != nil {
		return err
	}
	d.enabled = true
	return nil
}

func (d *sysfsPWM) SetDutyPercent(p float64) error {
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}

	if d.periodNS == 0 {
		// Conservative default if SetFrequencyHz wasn't called.
		d.periodNS = 1_000_000_000 / 64_000
	}

	duty := uint64(math.Round(float64(d.periodNS) * (p / 100.0)))
	if duty > d.periodNS {
		duty = d.periodNS
	}
	if err := d.writeUint("duty_cycle", duty); err != nil {
		return err
	}

	if !d.enabled {
		_ = d.writeBool("enable", true)
		d.enabled = true
	}
	return nil
}

func (d *sysfsPWM) writeUint(name string, v uint64) error {
	p := filepath.Join(d.pwmPath, name)
	return writeSysfs(p, strconv.FormatUint(v, 10))
}

func (d *sysfsPWM) writeBool(name string, v bool) error {
	p := filepath.Join(d.pwmPath, name)
	val := "0"
	if v {
		val = "1"
	}
	return writeSysfs(p, val)
}

func writeSysfs(path string, value string) error {
	// Use O_WRONLY without O_TRUNC/O_CREATE.
	// Some sysfs attributes reject truncation flags even when mode bits allow writes,
	// resulting in confusing EACCES/EPERM at open() time.
	// Additionally: immediately after exporting a PWM channel, the kernel creates
	// new sysfs files and udev may adjust permissions asynchronously. On some
	// systems there's a short window where open() returns EACCES or ENOENT even
	// though the steady-state permissions are correct.
	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			lastErr = err
			if time.Now().Before(deadline) && isRetryableSysfsErr(err) {
				time.Sleep(25 * time.Millisecond)
				continue
			}
			return err
		}
		_, werr := f.WriteString(value)
		cerr := f.Close()
		if werr == nil && cerr == nil {
			return nil
		}
		if werr != nil {
			lastErr = werr
		} else {
			lastErr = cerr
		}
		if time.Now().Before(deadline) && isRetryableSysfsErr(lastErr) {
			time.Sleep(25 * time.Millisecond)
			continue
		}
		if werr != nil && cerr != nil {
			return errors.Join(werr, cerr)
		}
		if werr != nil {
			return werr
		}
		return cerr
	}
}

func isRetryableSysfsErr(err error) bool {
	return os.IsPermission(err) || os.IsNotExist(err) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ENOENT)
}

func readInt(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}
