//go:build linux

package web

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func splitNMCLITerseLine(line string) []string {
	// nmcli -t output uses ':' separators and escapes ':' and '\\' with a backslash.
	// Example: "My\\:SSID:70:WPA2" means SSID "My:SSID".
	fields := make([]string, 0, 4)
	var b strings.Builder
	b.Grow(len(line))
	escaped := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == ':' {
			fields = append(fields, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	if escaped {
		// Trailing backslash; keep it.
		b.WriteByte('\\')
	}
	fields = append(fields, b.String())
	return fields
}

func scanWiFiNetworks(ctx context.Context, iface string) ([]WiFiScanNetwork, error) {
	// Scanning can take a while on some chipsets/drivers; keep this bounded but
	// long enough to avoid spurious cancellations.
	cmdCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	// Use terse output. Older nmcli versions don't support --separator;
	// SSIDs containing ':' are escaped as '\\:' in -t mode.
	args := []string{"-t", "-f", "SSID,SIGNAL,SECURITY", "dev", "wifi", "list", "--rescan", "yes"}
	if iface != "" {
		args = append(args, "ifname", iface)
	}

	cmd := exec.CommandContext(cmdCtx, "nmcli", args...)
	out, err := cmd.Output()
	if err != nil {
		if cmdCtx.Err() != nil {
			// When the context is canceled, exec.CommandContext may report "signal: killed".
			return nil, fmt.Errorf("nmcli scan timed out")
		}
		// Include stderr in error when available.
		if ee, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(ee.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("nmcli failed: %s", stderr)
			}
		}
		return nil, fmt.Errorf("nmcli failed: %v", err)
	}

	best := map[string]WiFiScanNetwork{}
	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() {
		line := strings.TrimRight(s.Text(), "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := splitNMCLITerseLine(line)
		if len(parts) < 1 {
			continue
		}
		ssid := strings.TrimSpace(parts[0])
		if ssid == "" {
			// Hidden or blank.
			continue
		}

		signal := 0
		security := ""
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				signal = n
			}
		}
		if len(parts) >= 3 {
			security = strings.TrimSpace(parts[2])
		}

		n := WiFiScanNetwork{SSID: ssid, Signal: signal, Security: security}
		prev, ok := best[ssid]
		if !ok || n.Signal > prev.Signal {
			best[ssid] = n
		} else if ok {
			// Keep any security info if the stronger signal entry didn't have it.
			if prev.Security == "" && n.Security != "" {
				prev.Security = n.Security
				best[ssid] = prev
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("nmcli parse failed: %v", err)
	}

	outNets := make([]WiFiScanNetwork, 0, len(best))
	for _, v := range best {
		outNets = append(outNets, v)
	}
	return outNets, nil
}
