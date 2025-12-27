//go:build linux

package web

import (
	"net"
	"sort"
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

func snapshotNetwork(_ time.Time) *NetworkSnapshot {
	addrs := localInterfaceAddrs()
	return &NetworkSnapshot{LocalAddrs: addrs}
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
