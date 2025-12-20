# Stratux-NG (Next Gen)

Stratux-NG is a modern, Raspberry Pi–focused, 64-bit-first avionics data appliance inspired by Stratux, designed to run on Raspberry Pi 3/4/5 and provide **GDL90** traffic/weather-style outputs over Wi‑Fi to EFB apps.

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

### Quick start (planned)
- `go run ./cmd/stratux-ng --config ./dev.yaml`
- Connect your tablet/phone to the Pi Wi‑Fi and point the EFB at GDL90 (per-app instructions)

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
- [ ] Record/replay mode for decoder feeds (for repeatable testing)
- [ ] Raspberry Pi image build pipeline (pi-gen or equivalent)
- [ ] Hardware integration: SDR 1090, SDR 978, GPS, AHRS