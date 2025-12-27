# ForeFlight setup

This guide aims to be exact, but ForeFlight UI labels can vary by iOS version and ForeFlight build. If any menu labels differ on your device, tell me what you see and I’ll adjust the doc.

## Network

- For broadcast-style setups, point `gdl90.dest` at your subnet broadcast (example: `192.168.10.255:4000`).
- For per-device delivery, set `gdl90.dest: "<iPad_or_iPhone_IP>:4000"`.

## ForeFlight steps (typical)

1) Connect the iPad/iPhone to the same IP network as Stratux-NG.
2) On iOS, ensure ForeFlight has **Local Network** permission enabled.
3) In ForeFlight:
   - Open **More** → **Devices**.
   - Look for a device entry that indicates a GDL90/Stratux-style receiver.
4) Verify:
   - GPS position becomes valid
   - Traffic appears
   - Device status shows receiving data

## If it doesn’t show up

- Run listen mode on the sender machine to confirm valid frames:
  - `go run ./cmd/stratux-ng --listen --listen-addr :4000`
- If broadcasts are filtered on your network, switch to unicast targeting your iPad/iPhone.
- Disable iOS VPN and any “Private Relay”/filtering features temporarily.
