# Stratux-NG (Next Gen)

Stratux-NG is a modern, Raspberry Pi–focused, 64-bit-first avionics data appliance inspired by Stratux, designed to run on Raspberry Pi 3/4/5 and provide **GDL90** traffic/weather-style outputs over UDP to EFB apps.

## Status

This repository is in active bring-up, but the core “useful on real hardware” loop is now working.

Working now:
- **GDL90 over UDP** (heartbeat + ownship + traffic + device ID + Stratux heartbeat)
- **Web UI + status API** (including decoder health)
- **AHRS (ICM-20948 + BMP280)** + **fan control**
- **1090** ingest from FlightAware `dump1090-fa` (`aircraft.json` polling) → **real GDL90 Traffic (0x14)**
- **978** ingest from `dump978-fa` (JSON/NDJSON over TCP) → **real GDL90 Traffic (0x14)**
- **978** uplink relay from `dump978-fa` raw TCP (`--raw-port`) → **GDL90 Uplink (0x07)** (EFB weather)

Not yet:
- “Pretty” decoded FIS-B product visualization in the web UI (later milestone)
- “Towers” page (UAT ground stations / GBTs): the UI entry exists but is currently a placeholder; implementing this requires decoding/deriving ground-station identity + signal stats from 978 UAT uplinks and exposing it via the status/API.
- Flashable SD image build pipeline (pi-gen stage implementation; see [docs/pi-gen.md](docs/pi-gen.md))

Web UI note:
- The web UI is still **work in progress**. Expect ongoing iteration (layout/navigation polish, additional pages for traffic/weather/towers parity, and more detailed status/config tooling).

This is a **new implementation** (new repository) with a modular architecture and reproducible builds, intended to support:

- **SDR inputs**
  - **1090 MHz ADS‑B / Mode S** via external decoder (e.g., FlightAware `dump1090-fa`)
  - **978 MHz UAT** via external decoder (e.g., `dump978`)
  - Support for “Nano 2/3” RTL-SDR-class devices and Stratux-compatible hardware
- **Sensors**
  - **GPS** (USB/serial; NMEA RMC+GGA supported)
  - **AHRS/IMU** (Stratux AHRS 2.0–class I2C sensors: ICM-20948 + BMP280)
- **Outputs**
  - **GDL90 over UDP** for EFB compatibility (initial focus: **Garmin Pilot** and **enRoute Flight Navigation**; enRoute will be primary test target early)

## Goals

### MVP (hardware-independent)
1. Run on **Raspberry Pi OS 64-bit (arm64)** on Pi 5 (and compatible with Pi 3/4).
2. Produce valid **GDL90** output over UDP from live ownship + traffic data.
3. Provide a minimal **HTTP API + web UI** for status and configuration.
4. Provide a path to building a **bootable image** (later milestone).

### Hardware integration (next)
- Ingest 1090 data from FlightAware `dump1090-fa` (done)
- Ingest 978 data from `dump978-fa` (done)
- GPS ingestion (gpsd integration) (done)
- Process supervision (restart decoders, health checks, logging) (done)

## Architecture (high level)

- `stratux-ng` (Go) is the core:
  - starts/configures inputs (dump1090-fa, dump978, gps, ahrs)
  - maintains an in-memory “traffic + ownship” state
  - outputs **GDL90 UDP**
  - serves an HTTP API + web UI for status/config

- External decoders are treated as data sources:
  - FlightAware `dump1090-fa` for 1090 MHz
  - `dump978` / `dump978-fa` for 978 MHz

Decoder I/O convention:
- 1090 recommended: `dump1090-fa --write-json ...` (Stratux-NG polls `aircraft.json`)
- 978 traffic recommended: `dump978-fa --json-port ...` (Stratux-NG ingests NDJSON over TCP)
- 978 weather recommended: `dump978-fa --raw-port ...` (Stratux-NG relays uplinks as GDL90 message `0x07`)

SDR serial tag note (Stratux-style):
- Many Stratux-tagged RTL-SDRs report USB serial strings like `stx:1090:0` and `stx:978:0` (as seen in `lsusb -v`).
- Use the exact string reported by your dongle when passing `dump1090-fa --device-type soapy --device driver=rtlsdr,serial=<serial>` or `dump978-fa --sdr driver=rtlsdr,serial=<serial>`.

## Wi-Fi Configuration

Stratux-NG supports a dual-mode Wi-Fi configuration on Raspberry Pi hardware (using the single built-in Wi-Fi interface):

1.  **Access Point (AP)**: Creates a hotspot (default SSID: `stratux`, no password) for EFBs to connect to.
2.  **Client Mode**: Simultaneously connects to an external Wi-Fi network (e.g., a phone hotspot) to provide internet access to connected devices.

This allows your EFB to receive GDL90 data from Stratux-NG while maintaining internet connectivity for weather updates, charts, etc.

Configuration is available via the Web UI (`Settings` -> `Wi-Fi`) or the API.

### Wi-Fi Prerequisites (Built-in)

The Stratux-NG image build process automatically handles the necessary system configuration for Wi-Fi operation. This includes:

1.  **IP Forwarding**: Enabled via `/etc/sysctl.d/90-router.conf` (`net.ipv4.ip_forward=1`) to allow traffic routing between the AP and Client interfaces.
2.  **NetworkManager**: Installed and used to manage Wi-Fi connections and the Access Point.
3.  **iw**: Installed for low-level wireless configuration.

## Development (Raspberry Pi 3/4/5 + VS Code)

### Quick start

Run Stratux-NG (broadcasts framed GDL90 over UDP):

- `go run ./cmd/stratux-ng --config ./config.yaml`

Config loading:
- If `--config` is not provided, Stratux-NG loads `/data/stratux-ng/config.yaml`.
- You can also set `STRATUX_NG_CONFIG` to a path (no CLI flag needed).

Web UI/status API:

- Web UI is enabled by default.
- Default listen address is `:80`.
- Browse to `http://<pi-ip>/` (or whatever `web.listen` you choose)

Port 80 note (Linux): binding to ports <1024 usually requires root or capabilities.
If you see a "permission denied" error when using `:80`, fix it by either:

- Running as root (this is what the provided systemd unit does), or
- Granting the binary `CAP_NET_BIND_SERVICE`:
  - `sudo setcap 'cap_net_bind_service=+ep' $(readlink -f ./stratux-ng)`
  - (systemd) set `AmbientCapabilities=CAP_NET_BIND_SERVICE` in the unit.

Web UI notes:
- Mobile-first layout intended for phone/tablet use.
- Bottom navigation switches between: Attitude, Radar, Map (placeholders for now).
- The menu button opens a small “More” drawer.

### Installing decoders (Raspberry Pi OS trixie, arm64)

Stratux-NG can either:
- **Supervise decoders itself** (recommended for development; simplest: one process), or
- **Connect to external decoders** (if you run `dump1090-fa` / `dump978-fa` as separate systemd services)

This repo targets **Raspberry Pi OS 64-bit (arm64)** on **Pi 3/4/5**.
Quick check: `uname -m` should print `aarch64`.

1) Install SDR + build dependencies:

```
sudo apt update
sudo apt install -y git build-essential cmake pkg-config \
  libusb-1.0-0-dev zlib1g-dev libncurses-dev \
  libboost-all-dev rtl-sdr soapysdr-tools soapysdr-module-rtlsdr libsoapysdr-dev
```

2) Prevent “device busy” (DVB driver grabbing RTL-SDR dongles):

```
echo 'blacklist dvb_usb_rtl28xxu' | sudo tee /etc/modprobe.d/rtl-sdr-blacklist.conf
sudo reboot
```

3) Build/install FlightAware `dump1090-fa` (1090):

```
cd ~
git clone https://github.com/flightaware/dump1090.git
cd dump1090
make -j"$(nproc)"

# Install with a stable name used by the default config.yaml:
sudo install -m 755 dump1090 /usr/local/bin/dump1090-fa
```

4) Build/install FlightAware `dump978` (build target: `dump978-fa`):

```
cd ~
git clone https://github.com/flightaware/dump978.git
cd dump978
make -j"$(nproc)" dump978-fa

# dump978 does not currently ship a "make install" target; install the binary manually:
sudo install -m 755 dump978-fa /usr/local/bin/dump978-fa
```

5) Validate decoder output paths/ports (defaults used by [config.yaml](config.yaml)):

```
# 1090 aircraft.json (dump1090-fa)
# Note: this file only exists after dump1090-fa is running.
# If you're using the provided systemd unit, systemd creates /run/dump1090-fa via RuntimeDirectory.
ls -l /run/dump1090-fa/aircraft.json
head /run/dump1090-fa/aircraft.json

# 978 JSON/NDJSON
nc 127.0.0.1 30978 | head
```

### Appliance / SD image build checklist

When you move from development (`go run ...`) to a flashable SD image, these are the practical “make it work every boot” steps:

- Persistent config path
  - Mount an ext4 partition at `/data`
  - Put config at `/data/stratux-ng/config.yaml`
  - See: [docs/sd-image-persistence.md](docs/sd-image-persistence.md)
- Stable GPS device name (recommended)
  - Install the udev rule example: `configs/udev/99-stratux-gps.rules.example`
  - Configure `gps.device: /dev/stratux-gps` (avoid hard-coding `/dev/ttyACM0`)
- gpsd (recommended for “any random USB GPS”)
  - Install/enable `gpsd` in the image and set `gps.source: gpsd` (default address: `127.0.0.1:2947`)
- Service management (recommended)
  - Install and enable the systemd unit: `configs/systemd/stratux-ng.service.example`
  - Ensure the service can read serial devices (typically group `dialout` if running unprivileged)
- Port binding
  - The default `web.listen: :80` works if the service runs as root (as in the provided systemd unit). If you prefer running unprivileged, use capabilities (or systemd `AmbientCapabilities`).

For pi-gen image planning and what to bake into the SD image, see:

- [docs/pi-gen.md](docs/pi-gen.md)

### Record / replay (GDL90 output frames)

Stratux-NG can record the *framed* GDL90 UDP packets it emits, then replay them later for deterministic EFB testing (no SDR/GPS/AHRS required).

- Record:
  - Set `gdl90.record.enable: true` and `gdl90.record.path: ./gdl90-record.log`
- Replay:
  - Set `gdl90.replay.enable: true` and `gdl90.replay.path: ./gdl90-record.log`
  - Optional: `gdl90.replay.speed` (e.g., `2.0` for 2x) and `gdl90.replay.loop: true`

Notes:
- Record and replay are mutually exclusive.

### CLI overrides

You can override record/replay settings without editing YAML:

```
go run ./cmd/stratux-ng --record /tmp/gdl90.log
go run ./cmd/stratux-ng --replay /tmp/gdl90.log --replay-speed 2 --replay-loop
```

### Log summary

To inspect a log file (message ID histogram, duration, etc.):

```
go run ./cmd/stratux-ng --log-summary /tmp/gdl90.log
```
Log format (written by record mode):
- First line: `START`
- Then one frame per line: `<t_ns>,<hex>` where `t_ns` is nanoseconds since START and `<hex>` is the raw framed UDP payload.

## Prerequisites

- **Target OS:** Raspberry Pi OS 64-bit (arm64). Current dev target: **Pi OS trixie**.
- **Tooling:** Go toolchain (**Go 1.22+**), plus typical Pi utilities for networking/AP setup
- **Decoders (optional):** FlightAware `dump1090-fa` (1090) and `dump978-fa`/`dump978` (978)

### Install build toolchain (Raspberry Pi OS 64-bit)

The commands below install the tools needed to compile Stratux-NG directly on a Pi 3/4/5 running Pi OS trixie (arm64).

```
sudo apt update
sudo apt install -y build-essential git pkg-config golang-go

go version
```

The `golang-go` package is currently Go 1.22 on Raspberry Pi OS trixie, installs under `/usr/lib/go`, and lands `/usr/lib/go/bin` in the default system `PATH`. Because the binary is registered system-wide, `sudo` and non-root shells use the same Go toolchain without additional PATH tweaks. If your image has an older package, install the `backports` repo or fetch a newer Go release.

### Code quality (development)

- Format: `gofmt -w .`
- Tests: `go test ./...`
- Vet: `go vet ./...`
- Static analysis (optional but recommended):
  - Debian/Trixie: `go-staticcheck ./...` (package: `go-staticcheck`)
  - Or use the repo target: `make staticcheck`

## AHRS (ICM-20948 + BMP280)

Stratux-NG can read a Stratux AHRS 2.0–class board over I2C (typically `0x68` for the IMU and `0x77` for the baro) and feed roll/pitch + pressure altitude into the existing Stratux-like AHRS GDL90 messages.

- Enable I2C on the Pi and confirm the sensors appear (example scan): `sudo i2cdetect -y 1`
- Enable AHRS in your config (see [config.yaml](config.yaml)):
  - `ahrs.enable: true`
  - `ahrs.i2c_bus: 1`
  - `ahrs.imu_addr: 0x68`
  - `ahrs.baro_addr: 0x77`

Notes:
- IMU is required for attitude; baro is optional.
- If `ahrs.enable` is true and the IMU cannot be initialized, Stratux-NG continues running but marks AHRS invalid.
- If the baro (BMP280) is missing or not responding at startup, Stratux-NG continues running (IMU-only) and periodically re-attempts baro detection.
- Baro address: Stratux AHRS 2.0 boards commonly use `0x77` (and sometimes `0x76`). Stratux-NG will probe both `0x77` and `0x76` even if `ahrs.baro_addr` is set to just one.
- Initial bring-up computes roll/pitch from accelerometer (gravity vector). Heading fuses gyro yaw rate with GPS track until magnetometer integration is added.

GDL90 altitude semantics (Stratux-compatible):
- Ownship Report (0x0A) altitude is treated as **pressure altitude** when available.
- Ownship Geometric Altitude (0x0B) remains **geometric altitude (MSL)**.

Heading/yaw note (Stratux-like):
- The AHRS board can measure **yaw rate** (gyro Z), but a stable absolute **yaw/heading angle** generally requires an external reference.
- Upstream Stratux behavior is effectively “gyro for short-term dynamics (turns), GPS track as a heading reference when valid”; magnetometer heading is typically left invalid until calibration/fusion is implemented.
- GPS ingestion is supported (USB serial NMEA) and provides real track/groundspeed for heading reference.

## GPS (USB serial NMEA)

Stratux-NG can read a USB GPS that presents as a serial device (common for the Stratux GPYes 2.0 u-blox8). It parses NMEA **RMC** (lat/lon/groundspeed/track) and **GGA** (altitude).

- Plug in the GPS and look for `/dev/ttyACM0` (common) or `/dev/ttyUSB0`.
- Appliance/image recommendation (Stratux-like): install a udev rule that creates a stable symlink (so the device name doesn’t renumber across reboots).
  - This repo includes an example: `configs/udev/99-stratux-gps.rules.example`
  - After installing the rule, use `gps.device: /dev/stratux-gps`
- Enable GPS in [config.yaml](config.yaml):
  - `gps.enable: true`
  - optional: `gps.source: gpsd` (recommended for appliance images targeting “any random USB GPS”)
    - `gps.gpsd_addr: 127.0.0.1:2947`
  - optional: `gps.device: /dev/stratux-gps` (recommended for images)
    - If omitted, Stratux-NG auto-detects `/dev/ttyACM*`/`/dev/ttyUSB*`
  - optional: `gps.baud: 9600`

Notes on `gpsd`:
- `gpsd` is not required for known-good GPS hardware, but it improves plug-and-play compatibility across varied USB GPS devices.
- Stratux-NG’s `gpsd` mode consumes gpsd JSON reports (TPV/SKY) and maps them to the same ownship/status fields.

Appliance/image note:
- On Raspberry Pi OS / Debian-based images, `gpsd` is typically run via systemd (often socket-activated).
- This repo includes a small systemd drop-in example to order Stratux-NG after gpsd: `configs/systemd/stratux-ng-gpsd.conf.example`

Linux permission note: if Stratux-NG logs an “open failed (permission denied)” for the device, grant access to the serial device (commonly by adding the service user to the `dialout` group, or by running the service with appropriate permissions).

Calibration + orientation (Stratux AHRS 2.0 style):
- **Set Level**: cages roll/pitch so the current attitude becomes (0,0).
- **Zero Drift**: estimates stationary gyro bias over ~2 seconds.
- **Set Forward (point nose-end up)**: point the end of the AHRS board that will face the aircraft nose up toward the sky; Stratux-NG records which axis is “forward”.
- **Finish Orientation (place level)**: place the board in its mounted in-flight orientation, keep it still for ~1s, and Stratux-NG captures gravity and builds a stable sensor→body mapping.
- Orientation is persisted to YAML when you press “Finish Orientation” in the Web UI.

Startup behavior:
- On startup, Stratux-NG does a best-effort Set Level + Zero Drift once the IMU has produced stable samples.

Verification (quick):
- Open the Status page and confirm:
  - “IMU detected” and “IMU working” are checked.
  - “Baro detected” and “Baro working” are checked.
  - `AHRS last error` is empty.
- Or check `GET /api/status` and confirm `ahrs.baro_working: true` and a recent `ahrs.baro_last_update_utc`.

Troubleshooting (AHRS not detected/working):
- Confirm I2C is enabled:
  - `sudo raspi-config` → Interface Options → I2C (recommended), or
  - ensure `/boot/firmware/config.txt` includes `dtparam=i2c_arm=on`.
- Confirm the bus exists: `ls -l /dev/i2c-*` (Stratux AHRS 2.0 is typically on `/dev/i2c-1`).
- Scan the bus: `sudo i2cdetect -y 1` and look for `68` (IMU) and `77` (baro).
- If the scan is empty:
  - double-check wiring/connector seating,
  - confirm the board is powered,
  - and check `dmesg | grep -i i2c` for controller/driver errors.

## Fan control (PWM on GPIO18)

Stratux-NG can drive a PWM cooling fan on Raspberry Pi using BCM GPIO 18 (same default pin and control behavior as upstream Stratux).

- Enable fan control in your config (see [config.yaml](config.yaml)):
  - `fan.enable: true`
  - `fan.pwm_pin: 18`
  - `fan.pwm_frequency: 64000`
  - `fan.temp_target_c: 50`
  - `fan.pwm_duty_min: 0`
  - `fan.update_interval: 5s`

Notes:
- If fan control fails to initialize (unsupported platform, permission issues, etc.), Stratux-NG continues running.
- When the fan control loop exits unexpectedly, it attempts a fail-safe “fan full on” duty.
Raspberry Pi OS notes:
- This uses Linux PWM via `/sys/class/pwm` (recommended for Pi 5 compatibility).
- Ensure the PWM overlay is enabled (example for Bookworm): add `dtoverlay=pwm-2chan` to `/boot/firmware/config.txt`.
- The service typically needs permission to write under `/sys/class/pwm` (run as root or grant appropriate access via systemd).

Pi 5 + Stratux AHRS 2.0 hat note:
- Some AHRS 2.0 fan circuits behave like a simple on/off “enable” (2-wire fan power switching). On Raspberry Pi 5, the PWM controller output may not drive that circuit as expected even when `/sys/class/pwm` appears to work.
- By default (when `fan.backend` is omitted/empty), Stratux-NG uses `auto`: it will choose `gpio` on Raspberry Pi 5 and `pwm` on Raspberry Pi 3/4.
- You can override explicitly with `fan.backend: pwm` or `fan.backend: gpio`.
- Quick check: if `gpioset -c gpiochip0 -t 5s,0 18=1` spins the fan but PWM does not, force `fan.backend: gpio`.

Troubleshooting (fan PWM not available):
- Confirm the overlay is active after reboot:
  - `ls -l /sys/class/pwm/` (should contain `pwmchip*`)
  - `cat /sys/class/pwm/pwmchip0/npwm` (should be non-zero)
- Stratux-NG currently supports `fan.pwm_pin: 18` (GPIO18 / PWM channel 0) via sysfs PWM.
- If `fan.last_error` mentions permissions, run the service as root or grant access to `/sys/class/pwm` via systemd.

Image build note (pi-gen):
- When we build a flashable SD image with pi-gen, bake `dtoverlay=pwm-2chan` into the image’s boot config by ensuring the generated `/boot/firmware/config.txt` includes that line.

## Prebuilt SD image (persistence)

For power-loss resilience and SD-card write minimization strategies for a prebuilt SD image, see:

- [docs/sd-image-persistence.md](docs/sd-image-persistence.md)

Stratux-NG itself focuses on:

- Binding/broadcasting GDL90 UDP on the configured network interface (details configurable; exact ports/addresses TBD)
- Serving an HTTP API + minimal web UI (for status/config)

### SD image layout (planned)

The intended “flashable SD image” delivery is an appliance-style system:

- Binary: `/usr/local/bin/stratux-ng`
- systemd unit: `/etc/systemd/system/stratux-ng.service`
- Persistent data mount: `/data` (ext4)
- Live config (read/write): `/data/stratux-ng/config.yaml`

The example systemd unit sets `STRATUX_NG_CONFIG=/data/stratux-ng/config.yaml` so the service always loads the same config path.

## EFB compatibility

- Output format: **GDL90 over UDP**

### Compatibility targets

Stratux-NG aims to be compatible with EFBs that can consume Stratux-style GDL90 over UDP. In practice, behavior varies by app/version, so we bias toward matching Stratux quirks where known.

Common EFBs that typically support GDL90/Stratux-style receivers:

- enRoute Flight Navigation (primary early test target)
- Garmin Pilot
- ForeFlight
- Others vary (e.g., WingX, iFly EFB, Avare, OzRunways)

### Message set (current)

Stratux-NG currently emits these GDL90 message IDs:

- `0x00` Heartbeat
- `0x0A` Ownship Report
- `0x0B` Ownship Geometric Altitude
- `0x14` Traffic Report (decoder-ingested targets)
- `0x65` Device ID / Capabilities ("ForeFlight ID")
- `0xCC` Stratux Heartbeat

Per-app connection steps will be documented once defaults (UDP port/broadcast behavior) are finalized.

## EFB Setup + Testing Loop

### Current defaults (Stratux-NG)

- GDL90 UDP destination is configured via `gdl90.dest` in YAML.
- `config.yaml` in this repo defaults to broadcast: `192.168.10.255:4000` (adjust for your subnet)
- Message transport: UDP, framed GDL90 (with CRC + byte-stuffing)

Notes:
- Broadcast is typically the easiest choice on a dedicated subnet.
- For local testing on one machine, you can use unicast `127.0.0.1:4000`.

### Listen mode (local test)

Listen mode binds a local UDP socket and dumps received frames (message ID + CRC status) so you can verify what’s being sent.

Example: local loopback test in two terminals:

- Terminal A (listener):
  - `go run ./cmd/stratux-ng --listen --listen-addr :4000`
- Terminal B (sender, unicast to localhost):
  - Set `gdl90.dest: "127.0.0.1:4000"` in your config
  - `go run ./cmd/stratux-ng`

Optional: add `--listen-hex` to print raw packet bytes as hex.

### Per-EFB setup (to be confirmed)

EFB setup guides live in `docs/efb/`:

- General guidance (any EFB): [docs/efb/other-efbs.md](docs/efb/other-efbs.md)
- ForeFlight: [docs/efb/foreflight.md](docs/efb/foreflight.md)
- Garmin Pilot: [docs/efb/garmin-pilot.md](docs/efb/garmin-pilot.md)

## Configuration
Stratux-NG supports both:
- **Config file** (YAML) for headless provisioning
- **Web UI** for interactive changes (note: `web.listen` is configured before startup, not via the Web UI)

## Roadmap (initial milestones)
- [x] Core Go service skeleton + config
- [x] GDL90 encoder + UDP broadcaster
- [x] HTTP API + minimal UI
- [x] Process supervision + stream reconnect for `dump1090-fa` / `dump978-fa`
- [x] Record/replay mode for *GDL90 output frames* (repeatable EFB testing)
- [ ] Raspberry Pi image build pipeline (pi-gen or equivalent)
- [x] Hardware integration: SDR 1090, SDR 978, GPS, AHRS

## Contributing

If you want to help early, the highest-value next steps are:

- Establish the initial Go module + `cmd/stratux-ng` entrypoint
- Define the YAML config schema and defaults
- Implement a first-pass GDL90 encoder + UDP broadcaster

If you tell me your preferred direction (core Go skeleton vs networking/AP scripts vs UI/API), I can start by scaffolding that structure next.