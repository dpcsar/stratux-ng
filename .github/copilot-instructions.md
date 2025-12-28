# Copilot instructions (Stratux-NG)

## Big picture
- Pipeline: inputs (sim/scenario, GPS, AHRS, decoders) → in-memory ownship + traffic → outputs (UDP GDL90 + HTTP API/UI).
- Main loop is best-effort: GPS/AHRS/fan init failures log and continue (don’t crash the appliance).

## Where things live
- Runtime orchestration: `cmd/stratux-ng` (main loop + live runtime wiring).
- GDL90: package `gdl90` (CRC16 + byte-stuffing framing, message builders: heartbeat 0x00, ownship 0x0A/0x0B, traffic 0x14, uplink 0x07, ForeFlight 0x65, Stratux heartbeat 0xCC).
- Decoder ingest/supervision: package `decoder` (supervised child processes + TCP NDJSON/line clients).
- Traffic: package `traffic` (`Store` with TTL/eviction + parsers for dump1090-fa Stratux JSON stream and dump978).
- Web: package `web` (/api/status + strict /api/settings + embedded UI assets).
- Deterministic testing aids: package `sim` (scenario YAML v1 fixtures) and package `replay` (stable line-oriented GDL90 log format).

## Compatibility rules (EFBs are picky)
- Always use `gdl90.Frame()` / `gdl90.Unframe()` for correct CRC16 + byte-stuffing + 0x7E flags.
- Packing mirrors Stratux quirks for interoperability: truncate lat/lon/track (don’t round), keep sentinel values, and keep Traffic Report (0x14) packing consistent.
- Heartbeat cadence is ~1 Hz (configured by `gdl90.interval`).

## Decoder conventions
- 1090: consume dump1090-fa `--net-stratux-port` JSON stream → parse → upsert into `traffic.Store`.
- 978: NDJSON traffic over TCP; raw uplink lines → `ParseDump978RawUplinkLine` → relay as GDL90 uplink (0x07).
- Config changes should go through `config.DefaultAndValidate()`.

## Web/settings (strict by design)
- /api/settings POST requires the full schema and rejects unknown/duplicate keys; YAML writes are atomic (temp + rename).

## Dev workflow
- Go 1.22.
- `make test` / `make run` (CONFIG → STRATUX_NG_CONFIG) / `make build` / `make staticcheck`.

## Ground truth
- When changing GDL90 layouts/packing/cadence, compare against upstream Stratux (`stratux/stratux`), especially gen_gdl90, traffic, and gps logic.

## Target hardware
- Primary dev target: Raspberry Pi 5 (64-bit / arm64); must also work on Pi 3/4 (arm64).
- AHRS hardware class: “Stratux AHRS 2.0” style I2C sensors (IMU commonly at addr 0x68; baro often BMP280).
- Fan control is in-scope and should fail safe (do not crash main loop; prefer safe fallback behavior on init/read errors).

## SDRs (inputs)
- SDR class: RTL-SDR-family devices (Stratux “Nano 2/3” style) for 1090 MHz ADS-B and 978 MHz UAT.
- Selection convention: prefer programming/using the dongle USB serial string (Stratux-style tags often look like `stx:1090:0` / `stx:978:0`).
- Config wiring: SDR selection lives under `adsb1090.sdr` and `uat978.sdr` (prefer `serial_tag`; fall back to `index` or a stable `path`). Decoders are external; Stratux-NG supervises them and ingests via JSON file / NDJSON / raw TCP.
