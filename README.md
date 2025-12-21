# Stratux-NG (Next Gen)

Stratux-NG is a modern, Raspberry Pi–focused, 64-bit-first avionics data appliance inspired by Stratux, designed to run on Raspberry Pi 3/4/5 and provide **GDL90** traffic/weather-style outputs over Wi‑Fi to EFB apps.

## Status

This repository is currently in **early bring-up**. The simulator + UDP output path is working; hardware inputs (SDR/GPS/AHRS) are planned.

This is a **new implementation** (new repository) with a modular architecture and reproducible builds, intended to support:

- **SDR inputs**
  - **1090 MHz ADS‑B / Mode S** via external decoder (e.g., `readsb`)
  - **978 MHz UAT** via external decoder (e.g., `dump978`)
  - Support for “Nano 2/3” RTL-SDR-class devices and legacy Stratux-compatible hardware
- **Sensors**
  - **GPS** (USB/serial; NMEA or gpsd-backed)
  - **AHRS/IMU** (Stratux-class sensors; integration planned)
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

### Hardware integration (when devices arrive)
- Ingest 1090 data from `readsb`
- Ingest 978 data from `dump978`
- GPS ingestion (NMEA/gpsd)
- AHRS ingestion (IMU driver + attitude output as needed)
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

### Quick start

Run Stratux-NG (sends framed GDL90 over UDP from simulated ownship + traffic):

- `go run ./cmd/stratux-ng --config ./dev.yaml`

Then:

- Bring up the Wi‑Fi AP (see [docs/wifi-ap-hostapd-dnsmasq.md](docs/wifi-ap-hostapd-dnsmasq.md))
- Connect your tablet/phone (EFB device) to the Pi Wi‑Fi

### Record / replay (GDL90 output frames)

Stratux-NG can record the *framed* GDL90 UDP packets it emits, then replay them later for deterministic EFB testing (no SDR/GPS/AHRS required).

- Record (writes a simple timestamped hex log):
  - Set `gdl90.record.enable: true` and `gdl90.record.path: ./gdl90-record.log`
- Replay (re-sends the recorded framed packets over UDP):
  - Set `gdl90.replay.enable: true` and `gdl90.replay.path: ./gdl90-record.log`
  - Optional: `gdl90.replay.speed` (e.g., `2.0` for 2x) and `gdl90.replay.loop: true`

Notes:
- Record and replay are mutually exclusive.
- Recording is only supported in `gdl90.mode: gdl90` (not `mode: test`).

Log format (written by record mode):
- First line: `START`
- Then one frame per line: `<t_ns>,<hex>` where `t_ns` is nanoseconds since START and `<hex>` is the raw framed UDP payload.

## Prerequisites (planned)

- **Target OS:** Raspberry Pi OS 64-bit (arm64)
- **Tooling:** Go toolchain (version TBD), plus typical Pi utilities for networking/AP setup
- **Decoders (optional):** `readsb` (1090) and `dump978` (978) treated as external processes/data sources

## Networking / Wi‑Fi AP

Stratux-NG is intended to behave like an “appliance” on a Raspberry Pi: you power it on, connect your tablet/phone to its Wi‑Fi network, and your EFB receives **GDL90 over UDP**.

To keep networking reliable on Raspberry Pi, **AP configuration is host-managed** initially:

- **Option A:** `systemd` + `hostapd` + `dnsmasq`
- **Option B:** NetworkManager (if that fits your base image better)

Setup guide + templates:

- [docs/wifi-ap-hostapd-dnsmasq.md](docs/wifi-ap-hostapd-dnsmasq.md)
- `configs/wifi/hostapd.conf.example`
- `configs/wifi/dnsmasq-stratux-ng.conf.example`

Stratux-NG itself focuses on:

- Binding/broadcasting GDL90 UDP on the Pi’s Wi‑Fi interface (details configurable; exact ports/addresses TBD)
- Serving an HTTP API + minimal web UI (for status/config)

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

## Configuration
Stratux-NG supports both:
- **Config file** (YAML) for headless provisioning
- **Web UI** for interactive changes

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