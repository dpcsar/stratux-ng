# Garmin Pilot setup

Garmin Pilot UI labels differ between iOS/Android and versions. This guide is written to avoid guessing exact button text where it’s known to vary.

If you tell me:
- iOS or Android
- Garmin Pilot version
- the exact menu labels you see

…I can make this guide fully “click-by-click”.

## Network

- For broadcast deployments, set `gdl90.dest` to your subnet broadcast (example: `192.168.10.255:4000`).
- For per-device delivery, set `gdl90.dest: "<tablet_or_phone_IP>:4000"`.

## Garmin Pilot steps (generic, but reliable)

These are the same high-level steps on both platforms. Platform-specific notes are below.

1) Connect the device to the same IP network as Stratux-NG.
2) Ensure Garmin Pilot has the necessary network permissions (iOS/Android differ).
3) In Garmin Pilot, locate the **device/receiver** settings section and select an **external ADS‑B / GDL90 / Stratux-style** source if available.
4) Confirm status indicators show an external source connected and traffic/GPS data present.

### Weather (FIS-B) note

Stratux-NG relays **978 UAT uplinks** (FIS-B weather) the same way Stratux does: as **GDL90 Uplink (message `0x07`)** frames.

To enable weather relay, your 978 decoder should provide a raw uplink TCP stream:

- Run `dump978-fa` with `--raw-port <port>` (in addition to `--json-port <port>` for traffic).
- Configure the `uat978` decoder with either `raw_listen: <host:port>` or `raw_addr: <host:port>`.

If traffic works but weather does not, the most common cause is that `--raw-port` is not enabled or the `raw_*` endpoint isn’t configured.

### iOS notes

- If prompted, allow **Local Network** access for Garmin Pilot.
- If the connection never comes up, verify it’s enabled in iOS system settings for the app (iOS exposes a per-app Local Network toggle).
- If the app offers an **IP address / host** field for an external receiver, enter the iPhone/iPad IP and use port `4000`.

### Android notes

- If prompted, allow local network / nearby devices access.
- Some Android builds gate local network discovery behind **Location** permission/services. If Garmin Pilot can’t discover anything, temporarily allow Location (and ensure Location services are enabled) while you set up the connection.
- If the app offers an **IP address / host** field for an external receiver, enter the Android device IP and use port `4000`.

## Troubleshooting

- Use listen mode to confirm packets are arriving on the expected port.
- If broadcasts are filtered by your network, prefer unicast.
- If Garmin Pilot only supports a specific receiver type on your platform, tell me what choices it presents and I’ll map Stratux-NG to the closest supported option.

### Quick validation checklist

- Confirm `dump978-fa` is running and listening on both ports:
	- JSON/NDJSON port (traffic ingest)
	- raw port (uplink/weather relay)
- Confirm Stratux-NG status shows the 978 raw stream connected (see the web status page / status API).
- Confirm Garmin Pilot shows an external receiver connected and has time to download products (FIS-B weather can lag behind traffic).
