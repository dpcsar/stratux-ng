#!/bin/bash -e

# pi-gen custom stage: Stratux-NG
# This script runs during the pi-gen build and can both copy files into the image
# and run commands inside the target rootfs via on_chroot.

install -d "${ROOTFS_DIR}/usr/local/bin"
install -d "${ROOTFS_DIR}/etc/systemd/system"
install -d "${ROOTFS_DIR}/etc/udev/rules.d"
install -d "${ROOTFS_DIR}/data/stratux-ng"

install -m 0755 files/usr/local/bin/stratux-ng "${ROOTFS_DIR}/usr/local/bin/stratux-ng"
if [[ -f files/usr/local/bin/stratux-ng-wifi-apply ]]; then
  install -m 0755 files/usr/local/bin/stratux-ng-wifi-apply "${ROOTFS_DIR}/usr/local/bin/stratux-ng-wifi-apply"
fi
install -m 0644 files/etc/systemd/system/stratux-ng.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng.service"
install -m 0644 files/etc/udev/rules.d/99-stratux-gps.rules "${ROOTFS_DIR}/etc/udev/rules.d/99-stratux-gps.rules"
install -m 0644 files/data/stratux-ng/config.yaml "${ROOTFS_DIR}/data/stratux-ng/config.yaml"

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
EOF
