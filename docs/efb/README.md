# EFB setup (GDL90 / Stratux-style)

This folder contains **practical setup notes** for using Stratux-NG with Electronic Flight Bags (EFBs).

## Network assumptions

- **Pi AP subnet:** `192.168.10.1/24`
  - Broadcast: `192.168.10.255`
- **Home Wi‑Fi subnet:** `192.168.0.0/24`
  - Broadcast: `192.168.0.255`

## Stratux-NG defaults

- Protocol: **UDP**
- Payload: **framed GDL90** (0x7E flags, byte-stuffing, CRC16)
- Default port used in most EFB ecosystems: **4000/udp**
- Configured in YAML as `gdl90.dest`:
  - Broadcast example (Pi AP): `192.168.10.255:4000`
  - Unicast example (tablet/phone): `<EFB_IP>:4000`

## Guides

- ForeFlight: [foreflight.md](foreflight.md)
- Garmin Pilot: [garmin-pilot.md](garmin-pilot.md)
- Other EFBs (generic checklist): [other-efbs.md](other-efbs.md)

## Debugging tools

- Listen mode (local UDP sniffer):
  - `go run ./cmd/stratux-ng --listen --listen-addr :4000`
  - Add `--listen-hex` to dump raw packets.
