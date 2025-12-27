package wifi

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// EnsureAPInterface checks if the uap0 interface exists, and creates it if not.
// This requires root privileges.
func EnsureAPInterface() error {
	// Ensure wlan0 is up, as it's the physical device for uap0.
	// We also disable power save to prevent the AP from dropping.
	_ = exec.Command("ip", "link", "set", "wlan0", "up").Run()
	_ = exec.Command("iw", "dev", "wlan0", "set", "power_save", "off").Run()

	// Check if uap0 exists
	cmd := exec.Command("iw", "dev", "uap0", "info")
	if err := cmd.Run(); err == nil {
		return nil // Already exists
	}

	// Create uap0 with a distinct MAC address to avoid conflicts.
	// We derive it from wlan0's MAC but flip the locally administered bit.
	wlan0, err := net.InterfaceByName("wlan0")
	if err != nil {
		return fmt.Errorf("wlan0 not found: %v", err)
	}
	newMac := make(net.HardwareAddr, len(wlan0.HardwareAddr))
	copy(newMac, wlan0.HardwareAddr)
	if len(newMac) > 0 {
		newMac[0] ^= 0x02 // Flip locally administered bit
	}

	// iw dev wlan0 interface add uap0 type __ap addr <mac>
	cmd = exec.Command("iw", "dev", "wlan0", "interface", "add", "uap0", "type", "__ap", "addr", newMac.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create uap0: %v, output: %s", err, string(out))
	}

	return nil
}

// SetupAP configures the Access Point using NetworkManager.
func SetupAP(ssid, password, ip string) error {
	if err := EnsureAPInterface(); err != nil {
		return err
	}

	if ip == "" {
		ip = "192.168.10.1"
	}
	// Default to /24 if no mask provided
	if !strings.Contains(ip, "/") {
		ip = ip + "/24"
	}

	// Check if connection exists
	connName := "StratuxAP"

	// Delete existing connection if it exists to ensure clean state
	// We ignore errors here as it might not exist
	_ = exec.Command("nmcli", "con", "delete", connName).Run()

	// Create new connection
	// nmcli con add type wifi ifname uap0 con-name StratuxAP autoconnect yes ssid "SSID" ...
	args := []string{
		"con", "add", "type", "wifi", "ifname", "uap0", "con-name", connName,
		"autoconnect", "yes", "save", "yes",
		"ssid", ssid, "mode", "ap",
		"wifi.band", "bg", "wifi.channel", "6",
	}

	// Add security if password is provided
	if password != "" {
		args = append(args,
			"wifi-sec.key-mgmt", "wpa-psk",
			"wifi-sec.proto", "rsn",
			"wifi-sec.pairwise", "ccmp",
			"wifi-sec.group", "ccmp",
			"wifi-sec.psk", password,
		)
	}

	cmd := exec.Command("nmcli", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create AP connection: %v, output: %s", err, string(out))
	}

	// Set IP address
	// nmcli con modify StratuxAP ipv4.addresses 192.168.10.1/24 ipv4.method shared
	cmd = exec.Command("nmcli", "con", "modify", connName,
		"ipv4.addresses", ip,
		"ipv4.method", "shared")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set AP IP: %v, output: %s", err, string(out))
	}

	// Bring up connection
	cmd = exec.Command("nmcli", "con", "up", connName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring up AP: %v, output: %s", err, string(out))
	}

	return nil
}

// CalculateBroadcastAddress returns the broadcast address for a given IP/CIDR.
// If no CIDR is provided, /24 is assumed.
func CalculateBroadcastAddress(ipStr string) (string, error) {
	if !strings.Contains(ipStr, "/") {
		ipStr = ipStr + "/24"
	}
	ip, ipNet, err := net.ParseCIDR(ipStr)
	if err != nil {
		return "", err
	}

	// Calculate broadcast address
	// Broadcast = IP | ^Mask
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("only IPv4 supported")
	}
	mask := ipNet.Mask
	broadcast := make(net.IP, len(ip4))
	for i := 0; i < len(ip4); i++ {
		broadcast[i] = ip4[i] | ^mask[i]
	}

	return broadcast.String(), nil
}

// ConnectClient configures the Client connection (wlan0) to an external AP.
func ConnectClient(ssid, password string) error {
	// Ensure wlan0 is managed so NetworkManager can use it.
	// This overrides the default unmanaged state set in 99-unmanage-wlan0.conf.
	_ = exec.Command("nmcli", "dev", "set", "wlan0", "managed", "yes").Run()
	// Give NetworkManager a moment to recognize the device state change.
	time.Sleep(1 * time.Second)

	connName := "StratuxClient"

	// Delete existing connection to avoid duplicates
	_ = exec.Command("nmcli", "con", "delete", connName).Run()

	// Use 'device wifi connect' which is more robust than 'con add' + 'con up'
	// as it auto-detects security settings (WPA2/WPA3) and handles association.
	// nmcli device wifi connect "SSID" password "PASSWORD" ifname wlan0 name StratuxClient
	args := []string{
		"device", "wifi", "connect", ssid,
		"ifname", "wlan0",
		"name", connName,
	}
	if password != "" {
		args = append(args, "password", password)
	}

	cmd := exec.Command("nmcli", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to connect client: %v, output: %s", err, string(out))
	}

	return nil
}

type WiFiStatus struct {
	APSSID      string `json:"ap_ssid"`
	APIP        string `json:"ap_ip"`
	ClientSSID  string `json:"client_ssid"`
	ClientState string `json:"client_state"` // connected, connecting, disconnected
	ClientIP    string `json:"client_ip"`
}

func GetStatus() (WiFiStatus, error) {
	status := WiFiStatus{}

	// Get AP SSID
	// nmcli -g 802-11-wireless.ssid connection show StratuxAP
	cmd := exec.Command("nmcli", "-g", "802-11-wireless.ssid", "connection", "show", "StratuxAP")
	if out, err := cmd.Output(); err == nil {
		status.APSSID = strings.TrimSpace(string(out))
	}

	// Get AP IP
	// nmcli -g ipv4.addresses connection show StratuxAP
	cmd = exec.Command("nmcli", "-g", "ipv4.addresses", "connection", "show", "StratuxAP")
	if out, err := cmd.Output(); err == nil {
		status.APIP = strings.TrimSpace(string(out))
	}

	// Get Client Status
	// nmcli -g GENERAL.STATE,IP4.ADDRESS connection show StratuxClient
	// Note: This only works if the connection is active.
	// Better to check device status for wlan0

	// Let's check active connections on wlan0
	cmd = exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE,STATE", "con", "show", "--active")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.Split(line, ":")
			if len(parts) < 4 {
				continue
			}
			if parts[2] != "wlan0" || parts[1] != "802-11-wireless" {
				continue
			}
			ssid := lookupConnectionSSID(parts[0])
			if ssid == "" {
				ssid = parts[0]
			}
			status.ClientSSID = ssid
			status.ClientState = parts[3]
			break
		}
	}

	// If we found a client connection, get its IP
	if status.ClientState == "activated" {
		cmd = exec.Command("nmcli", "-g", "ip4.address", "dev", "show", "wlan0")
		if out, err := cmd.Output(); err == nil {
			status.ClientIP = strings.TrimSpace(string(out))
		}
	}

	return status, nil
}

func lookupConnectionSSID(connName string) string {
	if strings.TrimSpace(connName) == "" {
		return ""
	}
	cmd := exec.Command("nmcli", "-g", "802-11-wireless.ssid", "connection", "show", connName)
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}
