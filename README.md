# Stratux-NG (Next Gen)

Stratux-NG is a modern, Raspberry Pi–focused, 64-bit-first avionics data appliance inspired by Stratux, designed to run on Raspberry Pi 3/4/5 and provide **GDL90** traffic/weather-style outputs over Wi‑Fi to EFB apps.

## Status

This repository is in active bring-up. The simulator + UDP output path is working, and **Stratux AHRS 2.0–class AHRS + fan control are supported**.

This is a **new implementation** (new repository) with a modular architecture and reproducible builds, intended to support:

- **SDR inputs**
  - **1090 MHz ADS‑B / Mode S** via external decoder (e.g., `readsb`)
  - **978 MHz UAT** via external decoder (e.g., `dump978`)
  - Support for “Nano 2/3” RTL-SDR-class devices and Stratux-compatible hardware
- **Sensors**
  - **GPS** (USB/serial; NMEA or gpsd-backed; planned)
  - **AHRS/IMU** (Stratux AHRS 2.0–class I2C sensors: ICM-20948 + BMP280)
- **Outputs**
  - **GDL90 over UDP** for EFB compatibility (initial focus: **Garmin Pilot** and **enRoute Flight Navigation**; enRoute will be primary test target early)
- **Networking**
  - Stratux-like Wi‑Fi behavior (AP mode, known SSID, DHCP) suitable for an “appliance” image people can flash to an SD card

## Goals

### MVP (hardware-independent)
1. Run on **Raspberry Pi OS 64-bit (arm64)** on Pi 5 (and compatible with Pi 3/4).
2. Produce valid **GDL90** output over UDP from **simulated** ownship + traffic.
3. Provide a minimal **HTTP API + web UI** for status and configuration.
4. Provide a path to building a **bootable image** (later milestone).

### Hardware integration (next)
- Ingest 1090 data from `readsb`
- Ingest 978 data from `dump978`
- GPS ingestion (NMEA/gpsd)
- GPS ingestion
- Process supervision (restart decoders, health checks, logging)

## Architecture (high level)

- `stratux-ng` (Go) is the core:
  - starts/configures inputs (sim, readsb, dump978, gps, ahrs)
  - maintains an in-memory “traffic + ownship” state
  - outputs **GDL90 UDP**
  - serves an HTTP API + web UI for status/config

- External decoders are treated as data sources:
  - `readsb` for 1090 MHz
  - `dump978` for 978 MHz

Wi‑Fi AP configuration is initially handled on the host (systemd + hostapd/dnsmasq or NetworkManager), to keep hardware/network control robust and simple on Raspberry Pi.

## Development (Raspberry Pi 5 + VS Code)

You can develop without SDR/GPS/AHRS hardware using the built-in simulator:

- Simulated ownship (moving track)
- Simulated traffic targets
- GDL90 broadcast over UDP so EFBs can connect and display traffic/position

### Deterministic scenario scripts (repeatable EFB testing)

For precise, repeatable EFB behavior testing (edge cases, regression repros), you can run a deterministic “scenario script” instead of the procedural sim.

- Enable `sim.scenario.enable: true` in your YAML config
- Set `sim.scenario.path` to a scenario file (see `configs/scenarios/`)
- Optionally set `sim.scenario.start_time_utc` to a fixed RFC3339 time (defaults to `2020-01-01T00:00:00Z` when scenario is enabled)

Sample scripts:

- `configs/scenarios/edgecases.yaml` (near poles, high/negative altitude, zero ground speed, abrupt heading changes, higher traffic count)
- `configs/scenarios/heading-wrap.yaml` (isolated track/heading wrap test: 350 → 10 → 350)
- `configs/scenarios/altitude-invalid.yaml` (isolated altitude invalid/sentinel transitions)

### Quick start

Run Stratux-NG (sends framed GDL90 over UDP from simulated ownship + traffic):

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

- Running as root (simple but not ideal), or
- Granting the binary `CAP_NET_BIND_SERVICE`:
  - `sudo setcap 'cap_net_bind_service=+ep' $(readlink -f ./stratux-ng)`
  - (systemd) set `AmbientCapabilities=CAP_NET_BIND_SERVICE` in the unit.

Web UI notes:
- Mobile-first layout intended for phone/tablet use.
- Bottom navigation switches between: Attitude, Radar, Map (placeholders for now).
- The menu button opens a small “More” drawer.

Then:

- Bring up the Wi‑Fi AP (see [docs/wifi-ap-hostapd-dnsmasq.md](docs/wifi-ap-hostapd-dnsmasq.md))
- Connect your tablet/phone (EFB device) to the Pi Wi‑Fi

### Record / replay (GDL90 output frames)

Stratux-NG can record the *framed* GDL90 UDP packets it emits, then replay them later for deterministic EFB testing (no SDR/GPS/AHRS required).

- Record:
  - Set `gdl90.record.enable: true` and `gdl90.record.path: ./gdl90-record.log`
- Replay:
  - Set `gdl90.replay.enable: true` and `gdl90.replay.path: ./gdl90-record.log`
  - Optional: `gdl90.replay.speed` (e.g., `2.0` for 2x) and `gdl90.replay.loop: true`

Notes:
- Record and replay are mutually exclusive.
- When `sim.scenario.enable: true`, recording timestamps are derived from the scenario time base (relative to the first emitted frame), so replays preserve scenario timing.

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

## Prerequisites (planned)

- **Target OS:** Raspberry Pi OS 64-bit (arm64)
- **Tooling:** Go toolchain (version TBD), plus typical Pi utilities for networking/AP setup
- **Decoders (optional):** `readsb` (1090) and `dump978` (978) treated as external processes/data sources

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
- Initial bring-up computes roll/pitch from accelerometer (gravity vector). Heading remains derived from the simulator until GPS/magnetometer integration is added.

GDL90 altitude semantics (Stratux-compatible):
- Ownship Report (0x0A) altitude is treated as **pressure altitude** when available.
- Ownship Geometric Altitude (0x0B) remains **geometric altitude (MSL)**.

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

Troubleshooting (fan PWM not available):
- Confirm the overlay is active after reboot:
  - `ls -l /sys/class/pwm/` (should contain `pwmchip*`)
  - `cat /sys/class/pwm/pwmchip0/npwm` (should be non-zero)
- Stratux-NG currently supports `fan.pwm_pin: 18` (GPIO18 / PWM channel 0) via sysfs PWM.
- If `fan.last_error` mentions permissions, run the service as root or grant access to `/sys/class/pwm` via systemd.

Image build note (pi-gen):
- When we build a flashable SD image with pi-gen, bake `dtoverlay=pwm-2chan` into the image’s boot config by ensuring the generated `/boot/firmware/config.txt` includes that line.

## Networking / Wi‑Fi AP

Stratux-NG is intended to behave like an “appliance” on a Raspberry Pi: you power it on, connect your tablet/phone to its Wi‑Fi network, and your EFB receives **GDL90 over UDP**.

To keep networking reliable on Raspberry Pi, **AP configuration is host-managed** initially:


Setup guide + templates:

## Prebuilt SD image (persistence)

For power-loss resilience and SD-card write minimization strategies for a prebuilt SD image, see:

- [docs/sd-image-persistence.md](docs/sd-image-persistence.md)
- `configs/wifi/dnsmasq-stratux-ng.conf.example`

Stratux-NG itself focuses on:

- Binding/broadcasting GDL90 UDP on the Pi’s Wi‑Fi interface (details configurable; exact ports/addresses TBD)
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

When simulator is enabled, Stratux-NG currently emits these GDL90 message IDs:

- `0x00` Heartbeat
- `0x0A` Ownship Report
- `0x0B` Ownship Geometric Altitude
- `0x14` Traffic Report (simulated targets)
- `0x65` Device ID / Capabilities ("ForeFlight ID")
- `0xCC` Stratux Heartbeat

Per-app connection steps will be documented once defaults (UDP port/broadcast behavior) are finalized.

## EFB Setup + Testing Loop

### Current defaults (Stratux-NG)

- GDL90 UDP destination is configured via `gdl90.dest` in YAML.
- `config.yaml` defaults to broadcast: `192.168.10.255:4000`
- Message transport: UDP, framed GDL90 (with CRC + byte-stuffing)

Notes:
- Broadcast is typically the easiest choice on a Wi‑Fi AP subnet.
- For local testing on one machine, you can use unicast `127.0.0.1:4000`.

### Listen mode (no Wi‑Fi/AP required)

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

Reference subnets used in this repo’s default setup:

- Pi AP: `192.168.10.1/24` (broadcast `192.168.10.255`)
- Home Wi‑Fi: `192.168.0.0/24` (broadcast `192.168.0.255`)

## Configuration
Stratux-NG supports both:
- **Config file** (YAML) for headless provisioning
- **Web UI** for interactive changes (note: `web.listen` is configured before startup, not via the Web UI)

## Roadmap (initial milestones)
- [ ] Core Go service skeleton + config
- [ ] GDL90 encoder + UDP broadcaster
- [ ] Simulator input (ownship + traffic)
- [ ] HTTP API + minimal UI
- [ ] Process supervisor scaffolding for `readsb` / `dump978`
- [ ] Record/replay mode for decoder feeds (and/or importers) for repeatable testing
- [ ] Raspberry Pi image build pipeline (pi-gen or equivalent)
- [ ] Hardware integration: SDR 1090, SDR 978, GPS, AHRS

## Contributing

If you want to help early, the highest-value next steps are:

- Establish the initial Go module + `cmd/stratux-ng` entrypoint
- Define the YAML config schema and defaults
- Implement a minimal simulator producing ownship + a few traffic targets
- Implement a first-pass GDL90 encoder + UDP broadcaster

If you tell me your preferred direction (core Go skeleton vs networking/AP scripts vs UI/API), I can start by scaffolding that structure next.