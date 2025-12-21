# Other EFBs (generic setup)

This is a generic checklist for any EFB that supports **GDL90** or **Stratux-style** receivers.

## 1) Pick a network topology

### Option A: Pi AP (recommended)

- Connect the EFB device to the Pi’s Wi‑Fi.
- Use broadcast: `gdl90.dest: "192.168.10.255:4000"`.

### Option B: Home Wi‑Fi

- Ensure both the Pi and the EFB device are on the same Wi‑Fi.
- Prefer unicast to the EFB device IP (most reliable): `gdl90.dest: "<EFB_IP>:4000"`.
- If you need broadcast, your /24 broadcast is `192.168.0.255:4000`.
  - Caveat: some consumer routers/APs block broadcast/multicast between clients.

## 2) Configure Stratux-NG

- `gdl90.mode: gdl90`
- `gdl90.interval: 1s`
- `gdl90.dest` set per above

Run:
- `go run ./cmd/stratux-ng --config ./dev.yaml`

## 3) Configure the EFB

Look for receiver types like:
- “Stratux”
- “GDL90”
- “ADS-B (GDL 90)”
- “External receiver”

Then verify:
- GPS/position becomes valid
- Traffic targets appear
- (If the EFB supports it) attitude/AHRS shows connected

## 4) Verify data path without Wi‑Fi/AP

On the same machine:
- Terminal A: `go run ./cmd/stratux-ng --listen --listen-addr :4000`
- Terminal B: set `gdl90.dest: "127.0.0.1:4000"` and run Stratux-NG normally.

If listen mode shows `crc_ok=true` and message IDs like `0x00`, `0x0A`, `0x14`, you’re sending valid framed GDL90.

## Common pitfalls

- iOS “Local Network” permission denied (the app can’t see UDP).
- Wrong subnet broadcast address (especially on /23 vs /24).
- Router/client isolation blocks broadcast (home Wi‑Fi).
- VPN enabled on the tablet (routes traffic away).
- Another service already bound to port 4000 (listen mode helps confirm).
