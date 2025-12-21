# Copilot instructions (Stratux‑NG)

## Target hardware (current)
- Primary dev target: Raspberry Pi 5 (64-bit / arm64).
- Must also work on: Raspberry Pi 4 (64-bit / arm64) and Raspberry Pi 3 (64-bit / arm64).
- AHRS hardware: “Stratux AHRS 2.0” (the same AHRS + fan-control add-on used by upstream Stratux).
  - Implementation should mirror upstream Stratux’s I2C probing/behavior as closely as practical.
  - Upstream Stratux supports IMU detection on I2C address `0x68` (e.g., ICM-20948 and MPU-9250 family) and commonly uses an onboard pressure sensor (e.g., BMP280) when present.
  - Fan-control is in-scope for this project.
    - Upstream Stratux implements fan control as a separate component (`fancontrol_main/`); mirror that behavior and defaults as closely as practical.
    - Prefer a robust, Pi-compatible implementation (Pi 3/4/5 on 64-bit) with safe failover (e.g., default fan ON on failure).

## Ground truth / reference implementation
- When you need to confirm protocol details, message layouts, bit fields, default values, or ecosystem quirks, **search the upstream Stratux repository first**:
  - Use the GitHub repo search tool against `stratux/stratux`.
  - Prefer matching behavior observed in `main/gen_gdl90.go`, `main/traffic.go`, and `main/gps.go`.
  - Goal: maximize compatibility with EFBs that expect Stratux-like GDL90.

## How to search Stratux (practical)
- Start with these files (highest signal for interoperability):
  - `main/gen_gdl90.go` (all core GDL90 message builders, including heartbeat/ownship/FF ID)
  - `main/traffic.go` (traffic report 0x14 and related fields/edge cases)
  - `main/gps.go` (GPS validity logic, NACp/NIC behavior, AHRS GDL90)
- Use repo search with specific anchors rather than broad terms. Good queries:
  - Heartbeat flags/time encoding: `makeHeartbeat msg[1] msg[2] secondsSinceMidnightUTC`
  - Stratux heartbeat: `makeStratuxHeartbeat 0xCC protocolVers`
  - Ownship (0x0A): `makeOwnshipReport msg[12] msg[13] 0x0A`
  - Ownship geo altitude (0x0B): `makeOwnshipGeometricAltitudeReport 0x0B`
  - ForeFlight ID (0x65): `makeFFIDMessage 0x65 Capabilities mask`
  - Traffic report (0x14): `makeTrafficReportMsg 0x14 msg[13] msg[15]`
- When you find a message builder in Stratux, mirror its:
  - field packing (bit/byte layout),
  - “invalid/unavailable” sentinel values,
  - and whether it’s emitted every second vs. only when data is available.

## Scope
- We are modernizing Stratux concepts into a smaller, maintainable Go codebase.
- Keep changes minimal and focused on the requested feature.
- Avoid large refactors unless they directly enable the requested behavior.

## Protocol / compatibility expectations
- Assume clients may be strict:
  - GDL90 framing must include proper CRC16 and byte-stuffing.
  - Heartbeat (0x00) should be sent at ~1 Hz.
  - When we simulate GPS, set the heartbeat GPS/UTC-valid bits consistently.
  - Prefer emitting Ownship Report (0x0A) + Ownship Geo Altitude (0x0B) when presenting a “GPS valid” device.
  - Prefer mirroring Stratux quantization quirks (e.g., track/heading fields should truncate rather than round) to avoid wrap-around glitches near 360°.
- If adding or changing message types, field packing, sentinels, or emission cadence:
  - Add/extend at least one focused unit test in `internal/gdl90`.
  - Add/extend a lightweight “sender emits expected message IDs” test in `cmd/stratux-ng` when main-loop output behavior changes.

## Code quality / build hygiene
- Run `go test ./...` after code changes.
- If feasible, also run `go test -race ./...` on supported platforms. Note: on some Raspberry Pi / arm64 environments, ThreadSanitizer can fail with errors like `ThreadSanitizer: unsupported VMA range`; treat that as an environment limitation unless there’s other evidence of a race.
- Keep code formatted with `gofmt`.
- Avoid adding new dependencies unless there is a clear benefit.
- Prefer tests that do not require network access (unit tests and small integration-ish tests that operate on frames in-memory).

## Determinism / record-replay
- For deterministic scenario testing, be careful about time bases:
  - Scenario mode uses a fixed UTC timeline; recording/replay should derive timestamps from the provided frame time (relative to first frame) rather than wall-clock `time.Now()`.
  - Add/adjust tests when changing record/replay timing semantics.

## Web/API safety
- Keep the web surface small and robust:
  - Add reasonable `http.Server` timeouts and header/body size limits.
  - When persisting YAML config changes, prefer atomic writes (write temp + rename) to avoid partial/corrupt configs on crash/power loss.

## Networking
- Default to simple, robust networking:
  - UDP destination must match the AP subnet (broadcast or explicit unicast).
  - Prefer configurability via `dev.yaml` and documented defaults.

## Documentation policy
- When code changes affect behavior, configuration, defaults, CLI flags, network setup, or interoperability, update docs in the same PR:
  - `README.md` for quick start, install/run, and high-level behavior.
  - `docs/` for detailed guides (e.g. Wi‑Fi/AP setup).
  - `configs/` templates when the recommended config changes.
- Keep docs consistent with shipped defaults (especially ports, IP ranges, and message types sent).

## Repo conventions
- Don’t add license headers unless explicitly requested.
- Keep documentation changes in `README.md` or `docs/`.
