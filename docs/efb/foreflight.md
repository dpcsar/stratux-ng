# ForeFlight setup

This guide aims to be exact, but ForeFlight UI labels can vary by iOS version and ForeFlight build. If any menu labels differ on your device, tell me what you see and I’ll adjust the doc.

## Network

### Pi AP (192.168.10.1/24)

- Set `gdl90.dest: "192.168.10.255:4000"`.

### Home Wi‑Fi (192.168.0.0/24)

- Prefer unicast: `gdl90.dest: "<your iPad/iPhone IP>:4000"`.
- Broadcast fallback: `gdl90.dest: "192.168.0.255:4000"`.

## ForeFlight steps (typical)

1) Connect the iPad/iPhone to the same Wi‑Fi as Stratux-NG (Pi AP or home Wi‑Fi).
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
- On home Wi‑Fi, switch from broadcast to unicast to your iPad/iPhone.
- Disable iOS VPN and any “Private Relay”/filtering features temporarily.
