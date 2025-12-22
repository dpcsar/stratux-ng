# gpsd (optional) for plug-and-play USB GPS

If your goal is “any random USB GPS” in an appliance image, running `gpsd` is recommended.

Stratux-NG supports two GPS ingestion modes:

- `gps.source: nmea` (default): read NMEA directly from a serial device
- `gps.source: gpsd`: read JSON reports from a local gpsd instance

## Why gpsd helps

- Better compatibility across varied USB GPS devices (baud rates, talkers, reconnect behavior)
- Provides richer fix metadata (mode/accuracy/satellites/hdop) without implementing every NMEA sentence

## Stratux-NG configuration

Set in your YAML:

- `gps.enable: true`
- `gps.source: gpsd`
- Optional: `gps.gpsd_addr: 127.0.0.1:2947`

## Systemd ordering (images)

On Raspberry Pi OS / Debian, `gpsd` is often socket-activated (`gpsd.socket`).

This repo includes a drop-in snippet to make `stratux-ng.service` start after `gpsd.socket`:

- `configs/systemd/stratux-ng-gpsd.conf.example`

Install example:

- `sudo mkdir -p /etc/systemd/system/stratux-ng.service.d`
- `sudo install -m 644 configs/systemd/stratux-ng-gpsd.conf.example /etc/systemd/system/stratux-ng.service.d/10-gpsd.conf`
- `sudo systemctl daemon-reload`
- `sudo systemctl restart stratux-ng`

## Raspberry Pi OS / Debian: install + enable (example)

Package/service names can vary slightly by distro release, but on Raspberry Pi OS / Debian these are typical:

- Install:
  - `sudo apt update`
  - `sudo apt install -y gpsd gpsd-clients`

- Enable gpsd:
  - Many images use socket activation:
    - `sudo systemctl enable --now gpsd.socket`
  - If your image uses a normal service instead:
    - `sudo systemctl enable --now gpsd.service`

- Verify gpsd is running/responding:
  - `gpspipe -w -n 5` (should print JSON including `TPV` and/or `SKY`)
  - `cgps -s` (interactive; press `q` to quit)

If you see “no devices”:

- Confirm the GPS enumerated:
  - `ls -l /dev/ttyACM* /dev/ttyUSB* 2>/dev/null`
- Check gpsd logs:
  - `journalctl -u gpsd.service -u gpsd.socket --no-pager | tail -n 200`

For appliance images, combining gpsd with a stable udev symlink (e.g. `/dev/stratux-gps`) is still recommended so your fallback path is deterministic.

## Troubleshooting

- Check gpsd is listening:
  - `ss -ltnp | grep 2947` (or `sudo ss -ltnp | grep 2947`)
- Check Stratux-NG status:
  - `GET /api/status` and look at the `gps.last_error` field

## PPS + time sync (future sensor fusion)

If you later need tighter timing (e.g. for sensor fusion / log correlation), add PPS + NTP discipline. This is optional and requires hardware support:

- Many USB GPS “pucks” do not expose PPS over USB. PPS is typically a separate pin (or sometimes mapped to a serial control line). You may need a GPS module with a PPS output wired to a Raspberry Pi GPIO.

Common high-level approach on Linux:

- Use `gpsd` for GNSS time-of-fix and position.
- Enable a PPS device (e.g. `/dev/pps0`) via kernel support (often `pps-gpio` for a GPIO pin).
- Configure `chrony` (or `ntpd`) to use both:
  - a coarse time source from gpsd (typically SHM)
  - a precise 1PPS source from `/dev/pps0`

Stratux-NG currently consumes GPS position/velocity and does not require system clock discipline to function, but disciplined time can materially improve cross-sensor timestamp alignment for more advanced pipelines.
