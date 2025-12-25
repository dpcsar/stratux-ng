# Wi‑Fi AP setup (hostapd + dnsmasq)

This guide sets up Raspberry Pi OS (64-bit) to run as a Wi‑Fi Access Point (AP) for Stratux-NG using **hostapd** (AP) + **dnsmasq** (DHCP/DNS).

It is intentionally **host-managed**: the OS owns Wi‑Fi configuration, while Stratux-NG only needs a reachable interface to send **GDL90 UDP** and serve HTTP.

## Stratux-NG defaults (first boot)

Stratux-NG defaults its Wi‑Fi AP network to `192.168.10.0/24` on first boot.

- AP subnet: `192.168.10.0/24`
- AP IP: `192.168.10.1`
- DHCP range: `192.168.10.50-192.168.10.150`
- Default GDL90 broadcast destination: `192.168.10.255:4000`

These values are editable in the web UI Settings page and stored in `config.yaml` under `wifi:` (`subnet_cidr`, `ap_ip`, `dhcp_start`, `dhcp_end`).

Stratux-NG also supports configuring an optional upstream Wi‑Fi (hotspot) connection and internet pass-through in `config.yaml` under `wifi:`:

- `uplink_enable`: whether to attempt connecting to an upstream Wi‑Fi network
- `client_networks`: list of SSIDs/passwords (password optional for open networks)
- `internet_passthrough_enable`: whether to enable IPv4 forwarding + NAT (masquerade) so AP clients can reach the internet

Important: Stratux-NG still does not configure the OS network stack itself — you must apply the same values to hostapd/dnsmasq/NetworkManager (examples below).

### Optional helper: generate dnsmasq config from `config.yaml`

Stratux-NG ships an optional helper binary that renders a dnsmasq snippet from the Settings-backed `wifi:` values:

```bash
# Print to stdout
STRATUX_NG_CONFIG=/data/stratux-ng/config.yaml \
  /usr/local/bin/stratux-ng-wifi-apply

# Write the dnsmasq config and restart dnsmasq (requires root)
sudo STRATUX_NG_CONFIG=/data/stratux-ng/config.yaml \
  /usr/local/bin/stratux-ng-wifi-apply \
  -out /etc/dnsmasq.d/stratux-ng.conf \
  -restart-dnsmasq

# Apply uplink + internet pass-through (nmcli + ip_forward + iptables NAT)
sudo STRATUX_NG_CONFIG=/data/stratux-ng/config.yaml \
  /usr/local/bin/stratux-ng-wifi-apply \
  -out /etc/dnsmasq.d/stratux-ng.conf \
  -restart-dnsmasq \
  -apply-internet
```

Example systemd oneshot units are provided at:

- `configs/systemd/stratux-ng-apply-wifi-dnsmasq.service.example`
- `configs/systemd/stratux-ng-apply-wifi-internet.service.example`

## What you want to achieve (two common modes)

### Mode 1: EFB connects, no internet required

- The Pi provides a Wi‑Fi AP.
- Your EFB connects to the Pi.
- Stratux-NG is reachable (HTTP + GDL90 UDP) even with **no uplink**.

This is the simplest and most robust configuration.

### Mode 2: EFB connects, Pi also has internet

- The Pi provides a Wi‑Fi AP for the EFB.
- The Pi also has an **uplink** to the internet (for updates, time sync, etc.).

In an aircraft you likely have **no Ethernet**, so the typical setup is:

- EFB connects to the Pi’s **AP**
- Pi connects to an upstream Wi‑Fi network as a **client (STA)** (phone hotspot, hangar Wi‑Fi, etc.)

This guide documents the **single-radio AP+STA** approach:

- `wlan0` stays as the **STA/uplink** interface
- a second, virtual interface (example: `ap0`) is created and used for the **AP**

Note: AP+STA requires driver support. If your hardware can’t do it, the fallback is a second Wi‑Fi adapter or Mode 1.

## Assumptions

- Raspberry Pi OS (arm64), with a Wi‑Fi interface named `wlan0`
- You want a private Wi‑Fi network (Stratux-NG default: `192.168.10.0/24`)
- You may want an upstream Wi‑Fi uplink on `wlan0`

Default values used in this guide:

- AP interface: `ap0` (virtual interface created from `wlan0`)
- Uplink interface: `wlan0`
- SSID: `stratux-ng`
- Security: open network (no WPA/WPA2)
- AP IP: `192.168.10.1/24`
- DHCP range: `192.168.10.50-192.168.10.150`

## 0) Check whether your Wi‑Fi supports AP+STA

Run:

```bash
iw list | sed -n '/valid interface combinations/,$p'
```

You are looking for a combination that allows **one managed (STA)** and **one AP** concurrently.

## 1) Install packages

```bash
sudo apt update
sudo apt install -y hostapd dnsmasq
```

Stop services while configuring:

```bash
sudo systemctl stop hostapd || true
sudo systemctl stop dnsmasq || true
```

## 2) Create the AP interface (`ap0`)

Create it immediately (for testing):

```bash
sudo iw dev wlan0 interface add ap0 type __ap
```

If that fails, your Wi‑Fi chipset/driver may not support AP+STA.

To create it automatically on boot, see the example templates:

- `configs/wifi/create-ap-interface.sh.example`
- `configs/wifi/iw-ap0.service.example`

One way to install them:

```bash
sudo install -m 755 -o root -g root configs/wifi/create-ap-interface.sh.example /usr/local/sbin/stratux-ng-create-ap0
sudo install -m 644 -o root -g root configs/wifi/iw-ap0.service.example /etc/systemd/system/stratux-ng-ap0.service
sudo systemctl daemon-reload
sudo systemctl enable --now stratux-ng-ap0.service
```

## 3) Give `ap0` a static IP

You need `ap0` to have a stable address (example: `192.168.10.1/24`). Choose ONE of the approaches below.

### Option A: NetworkManager (common on newer Raspberry Pi OS)

1. Check if NetworkManager is active:

```bash
systemctl is-active NetworkManager
```

2. Configure a static address for `ap0`:

```bash
sudo nmcli con add type ethernet ifname ap0 con-name stratux-ng-ap-ip ipv4.method manual ipv4.addresses 192.168.10.1/24 ipv6.method disabled
sudo nmcli con up stratux-ng-ap-ip
```

Notes:
- If `wlan0` already has a connection profile, you may want to modify it instead of adding a new one.

### Option B: dhcpcd (older/alternative setups)

Edit `/etc/dhcpcd.conf` and add:

```ini
interface ap0
    static ip_address=192.168.10.1/24
    nohook wpa_supplicant
```

Then restart dhcpcd:

```bash
sudo systemctl restart dhcpcd
```

## 4) Configure hostapd

Copy the example template from this repo and edit as needed:

- Template: `configs/wifi/hostapd.conf.example`

Install it:

```bash
sudo install -m 600 -o root -g root configs/wifi/hostapd.conf.example /etc/hostapd/hostapd.conf
```

Point hostapd at the config (Debian/RPi OS uses `/etc/default/hostapd`):

```bash
sudo sed -i 's|^#\?DAEMON_CONF=.*|DAEMON_CONF="/etc/hostapd/hostapd.conf"|' /etc/default/hostapd
```

## 4) Configure dnsmasq (DHCP)

Copy the example template from this repo and edit as needed:

- Template: `configs/wifi/dnsmasq-stratux-ng.conf.example`

Install it:

```bash
sudo install -m 644 -o root -g root configs/wifi/dnsmasq-stratux-ng.conf.example /etc/dnsmasq.d/stratux-ng.conf
```

(Optional) If you already have a lot of dnsmasq config, keep it simple and ensure only one DHCP server is active.

## 5) Enable IPv4 forwarding (optional but recommended)

If you want **Mode 2** (clients on the AP can reach the internet via the uplink `wlan0`), enable forwarding:

```bash
echo 'net.ipv4.ip_forward=1' | sudo tee /etc/sysctl.d/99-stratux-ng.conf
sudo sysctl --system
```

## 6) NAT/masquerade (optional)

If you want **Mode 2** (AP clients use the Pi’s uplink), you’ll also need NAT.

Raspberry Pi OS may use nftables under the hood; `iptables` may still work via compatibility. Pick ONE approach.

### Option A: iptables (simple)

```bash
sudo iptables -t nat -A POSTROUTING -o wlan0 -j MASQUERADE
sudo iptables -A FORWARD -i wlan0 -o ap0 -m state --state RELATED,ESTABLISHED -j ACCEPT
sudo iptables -A FORWARD -i ap0 -o wlan0 -j ACCEPT
```

To persist across reboots, install a persistence helper (choose one that matches your environment) or translate these rules into nftables.

### Option B: nftables (recommended if you already use it)

Create `/etc/nftables.conf` rules that masquerade traffic from `ap0` out `wlan0`. Since setups vary, this guide doesn’t enforce a specific nftables policy.

## 7) Start services

```bash
sudo systemctl unmask hostapd || true
sudo systemctl enable --now hostapd
sudo systemctl enable --now dnsmasq
```

## 8) Verify

1. Confirm `wlan0` has the AP IP:

```bash
ip addr show ap0
```

2. Check service logs:

```bash
sudo journalctl -u hostapd -n 100 --no-pager
sudo journalctl -u dnsmasq -n 100 --no-pager
```

3. From a phone/tablet:

- Join the SSID
- Confirm you receive a DHCP address in the expected range

## Stratux-NG integration notes

- Once Wi‑Fi AP is up, Stratux-NG should:
  - broadcast or unicast **GDL90 UDP** on the AP network
  - serve HTTP on the AP network

Exact ports/addresses will be documented when the initial Go service is implemented.

## WPS notes

You mentioned wanting WPS in both situations. There are two distinct uses:

1) **EFB -> Pi AP (hostapd WPS)**: helps a client join the Pi’s AP
2) **Pi -> upstream Wi‑Fi (wpa_supplicant WPS)**: helps the Pi join a hotspot/router

Notes:

- WPS is a security tradeoff; consider leaving it off and using a strong WPA2 passphrase.
- Many tablets (notably iOS) do not expose WPS in the Wi‑Fi UI.
- WPS for the **AP** requires WPA2; it does not work with an open (no-WPA) AP.

If you still want WPS:

- For **AP-side WPS**, first enable WPA2 in `configs/wifi/hostapd.conf.example` (WPS requires WPA2), then enable the WPS block and trigger:

```bash
sudo hostapd_cli -i ap0 wps_pbc
```

- For **uplink-side WPS** (Pi joining an upstream Wi‑Fi), wpa_supplicant supports:

```bash
sudo wpa_cli -i wlan0 wps_pbc
```

If you use NetworkManager for the uplink, use its connection UI/commands instead of `wpa_cli`.
