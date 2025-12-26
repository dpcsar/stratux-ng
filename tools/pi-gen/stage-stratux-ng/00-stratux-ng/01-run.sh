#!/bin/bash -e

# pi-gen custom stage: Stratux-NG
# This script runs during the pi-gen build and can both copy files into the image
# and run commands inside the target rootfs via on_chroot.

install -d "${ROOTFS_DIR}/usr/local/bin"
install -d "${ROOTFS_DIR}/usr/local/sbin"
install -d "${ROOTFS_DIR}/etc/systemd/system"
install -d "${ROOTFS_DIR}/etc/udev/rules.d"
install -d "${ROOTFS_DIR}/etc/hostapd"
install -d "${ROOTFS_DIR}/etc/default"
install -d "${ROOTFS_DIR}/etc/dnsmasq.d"
install -d "${ROOTFS_DIR}/etc/modules-load.d"
install -d "${ROOTFS_DIR}/etc/network"
install -d "${ROOTFS_DIR}/etc/network/interfaces.d"
install -d "${ROOTFS_DIR}/data/stratux-ng"

install -m 0755 files/usr/local/bin/stratux-ng "${ROOTFS_DIR}/usr/local/bin/stratux-ng"
if [[ -f files/usr/local/bin/stratux-ng-wifi-apply ]]; then
  install -m 0755 files/usr/local/bin/stratux-ng-wifi-apply "${ROOTFS_DIR}/usr/local/bin/stratux-ng-wifi-apply"
fi
if [[ -f files/usr/local/sbin/stratux-ng-create-ap0 ]]; then
  install -m 0755 files/usr/local/sbin/stratux-ng-create-ap0 "${ROOTFS_DIR}/usr/local/sbin/stratux-ng-create-ap0"
fi
install -m 0644 files/etc/systemd/system/stratux-ng.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng.service"
install -m 0644 files/etc/systemd/system/stratux-ng-apply-wifi-dnsmasq.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng-apply-wifi-dnsmasq.service"
install -m 0644 files/etc/systemd/system/stratux-ng-apply-wifi-internet.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng-apply-wifi-internet.service"
install -m 0644 files/etc/systemd/system/stratux-ng-create-ap0.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng-create-ap0.service"
install -m 0644 files/etc/systemd/system/stratux-ng-wifi.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng-wifi.service"
install -m 0644 files/etc/udev/rules.d/99-stratux-gps.rules "${ROOTFS_DIR}/etc/udev/rules.d/99-stratux-gps.rules"
install -m 0644 files/data/stratux-ng/config.yaml "${ROOTFS_DIR}/data/stratux-ng/config.yaml"
install -m 0644 files/etc/hostapd/hostapd.conf "${ROOTFS_DIR}/etc/hostapd/hostapd.conf"
install -m 0644 files/etc/default/hostapd "${ROOTFS_DIR}/etc/default/hostapd"
install -m 0644 files/etc/dnsmasq.d/stratux-ng.conf "${ROOTFS_DIR}/etc/dnsmasq.d/stratux-ng.conf"
install -m 0644 files/etc/modules-load.d/stratux-ng.conf "${ROOTFS_DIR}/etc/modules-load.d/stratux-ng.conf"
install -m 0644 files/etc/network/interfaces "${ROOTFS_DIR}/etc/network/interfaces"

# Blacklist the DVB driver that commonly grabs RTL-SDR devices.
cat >"${ROOTFS_DIR}/etc/modprobe.d/rtl-sdr-blacklist.conf" <<'EOF'
blacklist dvb_usb_rtl28xxu
EOF

on_chroot <<'EOF'
set -euo pipefail

# Create a dedicated service user/group.
if ! getent group stratuxng >/dev/null; then
  groupadd --system stratuxng
fi
if ! id stratuxng >/dev/null 2>&1; then
  useradd --system --gid stratuxng --home /nonexistent --shell /usr/sbin/nologin stratuxng
fi

# Ensure /data exists (real appliance images typically mount /data as a separate partition).
mkdir -p /data/stratux-ng
chown -R stratuxng:stratuxng /data/stratux-ng

# Enable service.
systemctl enable stratux-ng
systemctl enable stratux-ng-create-ap0.service
systemctl enable hostapd
systemctl enable dnsmasq
systemctl enable stratux-ng-wifi.service

systemctl disable --now wpa_supplicant.service || true
systemctl disable --now "wpa_supplicant@wlan0.service" || true
systemctl disable --now NetworkManager.service || true
systemctl disable --now dhcpcd.service || true

# Ensure I2C is enabled for AHRS hardware.
if ! grep -q '^dtparam=i2c_arm=on' /boot/firmware/config.txt 2>/dev/null; then
  printf '\n# Stratux-NG\ndtparam=i2c_arm=on\n' >> /boot/firmware/config.txt
fi
EOF
