# Stratux-NG

Stratux-NG is a Raspberry Pi–focused avionics data appliance inspired by Stratux.

It ingests ADS-B/UAT traffic (via external decoders), GPS and (optionally) AHRS sensors, then broadcasts **framed GDL90 over UDP** to Electronic Flight Bag (EFB) apps.

## What you get

- GDL90 over UDP (heartbeat, ownship, traffic, device ID, Stratux heartbeat)
- Web UI + `/api/status` for health and troubleshooting
- 1090 ADS-B ingest from FlightAware `dump1090-fa` (Stratux JSON stream)
- 978 UAT ingest from FlightAware `dump978-fa` (traffic) and raw uplink relay as GDL90 Uplink (weather)
- Optional AHRS (ICM-20948 + BMP280) and fan control

Project status: active development / bring-up. A prebuilt SD image pipeline is planned (see [docs/pi-gen.md](docs/pi-gen.md)).

## Quick start (run from source on a Pi)

This is the simplest way to get Stratux-NG running today.

### 1) Prereqs

- Raspberry Pi OS 64-bit (arm64) on Pi 3/4/5
- Go 1.22+

Install toolchain:

```
sudo apt update
sudo apt install -y git build-essential golang-go
go version
```

### 2) Configure

- Start from the repo’s [config.yaml](config.yaml).
- The most important settings for EFBs are under `gdl90`:
  - `gdl90.dest`: UDP destination (broadcast or unicast)
  - `gdl90.interval`: heartbeat cadence (default is 1s)

Default sample config broadcasts to `192.168.10.255:4000` (adjust the subnet to match your Wi‑Fi/AP network).

### 3) Run

```
go run ./cmd/stratux-ng --config ./config.yaml
```

Config path rules:
- If `--config` is not provided, Stratux-NG loads `/data/stratux-ng/config.yaml`.
- You can also set `STRATUX_NG_CONFIG` to a path.

Web UI:
- Default listen address is `:80`.
- Browse to `http://<pi-ip>/`.

Port 80 note: binding to ports <1024 typically requires root or capabilities. If you hit permissions issues, either run under the provided systemd unit (root) or change `web.listen` to a high port like `:8080`.

## Connect your EFB

1) Ensure your tablet/phone is on the same network as the Pi.
2) In your EFB, select a Stratux / GDL90 / “ADS-B receiver” type input.
3) Use UDP port `4000` unless you changed it.

Per‑EFB guides are in:
- [docs/efb/README.md](docs/efb/README.md)
- [docs/efb/foreflight.md](docs/efb/foreflight.md)
- [docs/efb/garmin-pilot.md](docs/efb/garmin-pilot.md)
- [docs/efb/other-efbs.md](docs/efb/other-efbs.md)

## Decoders (1090 / 978)

Stratux-NG expects external decoder binaries (it can supervise them, or you can run them separately).

- 1090 ADS-B: FlightAware `dump1090-fa`
- 978 UAT: FlightAware `dump978-fa`

If you don’t have decoders connected yet, Stratux-NG can still run (web UI + heartbeats), but you won’t see live traffic/weather.

## Troubleshooting

- Web UI: check the Status page.
- API: `GET /api/status` for detailed component health/errors.

Local UDP sanity check (sniff what you’re sending):

```
go run ./cmd/stratux-ng --listen --listen-addr :4000
```

## More docs

- gpsd (optional for plug-and-play USB GPS): [docs/gpsd.md](docs/gpsd.md)
- SD image persistence strategy: [docs/sd-image-persistence.md](docs/sd-image-persistence.md)
- pi-gen image plan: [docs/pi-gen.md](docs/pi-gen.md)

## Development

Developer notes (architecture, hardware bring-up details, decoder build instructions, record/replay, etc.) live in:

- [dev.md](dev.md)