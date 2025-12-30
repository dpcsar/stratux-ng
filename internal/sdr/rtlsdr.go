package sdr

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RTLSDRDevice describes one RTL-SDR-class device as enumerated by rtl_test.
//
// NOTE: Stratux-NG does not talk to SDR hardware directly; it only uses this
// for best-effort auto-configuration of external decoder CLI flags.
type RTLSDRDevice struct {
	Index  int
	Serial string
}

func IsAutoTag(tag string) bool {
	t := strings.TrimSpace(strings.ToLower(tag))
	return t == "" || t == "auto"
}

// DetectRTLSDRDevices enumerates RTL-SDR devices by shelling out to rtl_test.
//
// This is best-effort and intentionally permissive:
// - If rtl_test is missing or fails, an error is returned.
// - The caller should fall back to existing config/args.
func DetectRTLSDRDevices(ctx context.Context) ([]RTLSDRDevice, error) {
	// rtl_test can block briefly while opening devices; keep a short timeout.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rtl_test", "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If the command exists but returns non-zero, we still might have useful output.
		if len(out) == 0 {
			return nil, fmt.Errorf("rtl_test failed: %w", err)
		}
	}

	devs := ParseRTLTestOutput(string(out))
	if len(devs) == 0 {
		return nil, fmt.Errorf("no RTL-SDR devices found")
	}
	return devs, nil
}

var (
	rtlTestLineRE = regexp.MustCompile(`(?m)^\s*(\d+):\s+.*?\bSN:\s*([^\s]+)\s*$`)
)

// ParseRTLTestOutput extracts device indices + serials from rtl_test output.
func ParseRTLTestOutput(out string) []RTLSDRDevice {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	matches := rtlTestLineRE.FindAllStringSubmatch(out, -1)
	if len(matches) == 0 {
		return nil
	}
	devs := make([]RTLSDRDevice, 0, len(matches))
	seen := map[int]bool{}
	for _, m := range matches {
		idx, err := strconv.Atoi(strings.TrimSpace(m[1]))
		if err != nil {
			continue
		}
		if seen[idx] {
			continue
		}
		seen[idx] = true
		serial := strings.TrimSpace(m[2])
		devs = append(devs, RTLSDRDevice{Index: idx, Serial: serial})
	}
	sort.Slice(devs, func(i, j int) bool { return devs[i].Index < devs[j].Index })
	return devs
}

// AutoAssign1090And978 picks distinct devices for 1090 and 978.
//
// Heuristics:
// - Prefer serials containing "1090" / "978".
// - If only one matches, assign the other from remaining devices.
// - Otherwise, fall back to index order (0->1090, 1->978).
func AutoAssign1090And978(devs []RTLSDRDevice) (adsb1090 *RTLSDRDevice, uat978 *RTLSDRDevice) {
	if len(devs) == 0 {
		return nil, nil
	}

	findByHint := func(hint string) *RTLSDRDevice {
		hint = strings.ToLower(hint)
		for i := range devs {
			if strings.Contains(strings.ToLower(devs[i].Serial), hint) {
				return &devs[i]
			}
		}
		return nil
	}

	adsb := findByHint("1090")
	uat := findByHint("978")

	if adsb != nil && uat != nil {
		if adsb.Index != uat.Index {
			return adsb, uat
		}
		// Both hints picked the same device; fall back below.
	}

	// If one is known, pick the first remaining for the other.
	if adsb != nil {
		for i := range devs {
			if devs[i].Index != adsb.Index {
				return adsb, &devs[i]
			}
		}
		return adsb, nil
	}
	if uat != nil {
		for i := range devs {
			if devs[i].Index != uat.Index {
				return &devs[i], uat
			}
		}
		return nil, uat
	}

	// Fallback: stable order.
	adsb = &devs[0]
	if len(devs) >= 2 {
		uat = &devs[1]
	}
	return adsb, uat
}

// UpsertFlagValue ensures args contains flag set to value.
//
// Supports both "--flag value" and "--flag=value" forms.
func UpsertFlagValue(args []string, flag string, value string) []string {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return args
	}

	// First handle --flag=value.
	for i := range args {
		if strings.HasPrefix(args[i], flag+"=") {
			args[i] = flag + "=" + value
			return args
		}
	}

	// Then handle --flag value.
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			if i+1 < len(args) {
				args[i+1] = value
				return args
			}
			// Malformed; append value.
			return append(args, value)
		}
	}

	return append(args, flag, value)
}

// HasAnyFlag reports whether args contains any of the provided flags.
// It matches both "--flag" and "--flag=..." forms.
func HasAnyFlag(args []string, flags ...string) bool {
	set := map[string]struct{}{}
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f != "" {
			set[f] = struct{}{}
		}
	}
	for _, a := range args {
		if _, ok := set[a]; ok {
			return true
		}
		for f := range set {
			if strings.HasPrefix(a, f+"=") {
				return true
			}
		}
	}
	return false
}

// BuildDump978SoapyRTLSDRArg returns a SoapySDR-style device selector string.
//
// When serial is empty, it returns "driver=rtlsdr" which will let Soapy pick
// the first RTL-SDR device.
func BuildDump978SoapyRTLSDRArg(serial string) string {
	serial = strings.TrimSpace(serial)
	// "auto" is a Stratux-NG config convention; it is not a real RTL-SDR serial.
	// Passing serial=auto breaks SoapySDR selection.
	if IsAutoTag(serial) {
		return "driver=rtlsdr"
	}
	// Protect commas/whitespace (rare, but keep safe).
	serial = strings.ReplaceAll(serial, ",", "")
	serial = strings.TrimSpace(serial)
	return "driver=rtlsdr,serial=" + serial
}

// DebugFormatDevices formats devices for logging.
func DebugFormatDevices(devs []RTLSDRDevice) string {
	if len(devs) == 0 {
		return "[]"
	}
	var b bytes.Buffer
	b.WriteString("[")
	for i := range devs {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%d:%s", devs[i].Index, devs[i].Serial))
	}
	b.WriteString("]")
	return b.String()
}
