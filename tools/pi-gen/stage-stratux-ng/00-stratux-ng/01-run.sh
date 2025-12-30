#!/bin/bash -e

# pi-gen custom stage: Stratux-NG
# This script runs during the pi-gen build and can both copy files into the image
# and run commands inside the target rootfs via on_chroot.

install -d "${ROOTFS_DIR}/usr/local/bin"
install -d "${ROOTFS_DIR}/etc/systemd/system"
install -d "${ROOTFS_DIR}/etc/udev/rules.d"
install -d "${ROOTFS_DIR}/etc/modules-load.d"
install -d "${ROOTFS_DIR}/data/stratux-ng"

install -m 0755 files/usr/local/bin/stratux-ng "${ROOTFS_DIR}/usr/local/bin/stratux-ng"
install -m 0644 files/etc/systemd/system/stratux-ng.service "${ROOTFS_DIR}/etc/systemd/system/stratux-ng.service"
install -m 0644 files/etc/udev/rules.d/99-stratux-gps.rules "${ROOTFS_DIR}/etc/udev/rules.d/99-stratux-gps.rules"
install -m 0644 files/data/stratux-ng/config.yaml "${ROOTFS_DIR}/data/stratux-ng/config.yaml"
install -m 0644 files/etc/modules-load.d/stratux-ng.conf "${ROOTFS_DIR}/etc/modules-load.d/stratux-ng.conf"

# Enable IP forwarding for Wi-Fi repeater mode.
echo "net.ipv4.ip_forward=1" > "${ROOTFS_DIR}/etc/sysctl.d/90-router.conf"

# Blacklist the DVB driver that commonly grabs RTL-SDR devices.
cat >"${ROOTFS_DIR}/etc/modprobe.d/rtl-sdr-blacklist.conf" <<'EOF'
blacklist dvb_usb_rtl28xxu
EOF

on_chroot <<'EOF'
set -euo pipefail

# Ensure /data exists (real appliance images typically mount /data as a separate partition).
mkdir -p /data/stratux-ng

# Enable service.
systemctl enable stratux-ng

# Ensure I2C is enabled for AHRS hardware.
if ! grep -q '^dtparam=i2c_arm=on' /boot/firmware/config.txt 2>/dev/null; then
  printf '\n# Stratux-NG\ndtparam=i2c_arm=on\n' >> /boot/firmware/config.txt
fi

# Ensure hardware PWM is available for fan control on GPIO18/19.
# Without this overlay, /sys/class/pwm may be empty on Pi 3/4.
if ! grep -Eq '^dtoverlay=pwm-2chan\b' /boot/firmware/config.txt 2>/dev/null; then
  printf '\n# Stratux-NG\n# Enable hardware PWM for fan control (GPIO18/19)\ndtoverlay=pwm-2chan\n' >> /boot/firmware/config.txt
fi
EOF
