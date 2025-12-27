# EFB setup (GDL90 / Stratux-style)

This folder contains **practical setup notes** for using Stratux-NG with Electronic Flight Bags (EFBs).

## Stratux-NG defaults

- Protocol: **UDP**
- Payload: **framed GDL90** (0x7E flags, byte-stuffing, CRC16)
- Default port used in most EFB ecosystems: **4000/udp**
- Configured in YAML as `gdl90.dest`:
  - Broadcast example: `192.168.10.255:4000` (replace with your subnet broadcast)
  - Unicast example: `<EFB_IP>:4000`

## Altitude semantics (Stratux-compatible)

- Ownship Report (0x0A) altitude is **pressure altitude** when baro is available.
- Ownship Geometric Altitude (0x0B) is **geometric altitude (MSL)**.

## Guides

- ForeFlight: [foreflight.md](foreflight.md)
- Garmin Pilot: [garmin-pilot.md](garmin-pilot.md)
- Other EFBs (generic checklist): [other-efbs.md](other-efbs.md)

## Debugging tools

- Listen mode (local UDP sniffer):
  - `go run ./cmd/stratux-ng --listen --listen-addr :4000`
  - Add `--listen-hex` to dump raw packets.
