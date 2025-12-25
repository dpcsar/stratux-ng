//go:build linux

package web

import (
	"bufio"
	"errors"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func snapshotDisk(_ time.Time) *DiskSnapshot {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return &DiskSnapshot{LastError: err.Error()}
	}

	bsize := uint64(st.Bsize)
	total := st.Blocks * bsize
	free := st.Bfree * bsize
	avail := st.Bavail * bsize

	return &DiskSnapshot{
		RootPath:       "/",
		RootTotalBytes: total,
		RootFreeBytes:  free,
		RootAvailBytes: avail,
	}
}

func snapshotNetwork(nowUTC time.Time) *NetworkSnapshot {
	addrs := localInterfaceAddrs()
	clients, leasesFile, err := readDHCPClients(nowUTC)

	s := &NetworkSnapshot{
		LocalAddrs:     addrs,
		Clients:        clients,
		ClientsCount:   len(clients),
		DHCPLeasesFile: leasesFile,
	}
	if err != nil {
		s.LastError = err.Error()
	}
	return s
}

func localInterfaceAddrs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	out := make([]string, 0, 8)
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			var ipnet *net.IPNet
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
				ipnet = v
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			if ipnet != nil {
				out = append(out, iface.Name+": "+ipnet.String())
			} else {
				out = append(out, iface.Name+": "+ip4.String())
			}
		}
	}

	sort.Strings(out)
	return out
}

type dhcpLeaseCandidate struct {
	expires time.Time
	client  DHCPClientSnapshot
}

func readDHCPClients(nowUTC time.Time) ([]DHCPClientSnapshot, string, error) {
	paths := []string{
		"/var/lib/misc/dnsmasq.leases",
		"/var/lib/dnsmasq/dnsmasq.leases",
	}

	var lastErr error
	for _, p := range paths {
		clients, err := parseDNSMasqLeasesFile(p, nowUTC)
		if err == nil {
			return clients, p, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			lastErr = err
			continue
		}
		// If the file exists but can't be parsed, return error and the file.
		return nil, p, err
	}
	// If no lease file exists, AP/DHCP likely isn't enabled; that's not an error.
	if errors.Is(lastErr, os.ErrNotExist) {
		return nil, "", nil
	}
	return nil, "", lastErr
}

// parseDNSMasqLeasesFile parses dnsmasq's lease file:
//
//	expiry_epoch mac ip hostname clientid
func parseDNSMasqLeasesFile(path string, nowUTC time.Time) ([]DHCPClientSnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	candidates := make([]dhcpLeaseCandidate, 0, 16)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		expSec, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		expires := time.Unix(expSec, 0).UTC()
		// Only show active leases as "connected clients".
		if !expires.After(nowUTC.UTC()) {
			continue
		}

		mac := strings.ToLower(fields[1])
		ip := fields[2]
		host := fields[3]
		if host == "*" {
			host = ""
		}

		candidates = append(candidates, dhcpLeaseCandidate{
			expires: expires,
			client: DHCPClientSnapshot{
				MAC:        mac,
				IP:         ip,
				Hostname:   host,
				ExpiresUTC: expires.Format(time.RFC3339),
			},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(candidates, func(i, j int) bool {
		// Prefer hostname, then IP.
		hi := candidates[i].client.Hostname
		hj := candidates[j].client.Hostname
		if hi != hj {
			return hi < hj
		}
		return candidates[i].client.IP < candidates[j].client.IP
	})

	out := make([]DHCPClientSnapshot, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.client)
	}
	return out, nil
}
