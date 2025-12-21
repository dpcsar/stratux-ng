# Garmin Pilot setup

Garmin Pilot UI labels differ between iOS/Android and versions. This guide is written to avoid guessing exact button text where it’s known to vary.

If you tell me:
- iOS or Android
- Garmin Pilot version
- the exact menu labels you see

…I can make this guide fully “click-by-click”.

## Network

### Pi AP (192.168.10.1/24)

- Set `gdl90.dest: "192.168.10.255:4000"`.

### Home Wi‑Fi (192.168.0.0/24)

- Prefer unicast: `gdl90.dest: "<tablet/phone IP>:4000"`.
- Broadcast fallback: `gdl90.dest: "192.168.0.255:4000"`.

## Garmin Pilot steps (generic, but reliable)

These are the same high-level steps on both platforms. Platform-specific notes are below.

1) Connect the device to the same Wi‑Fi as Stratux-NG.
2) Ensure Garmin Pilot has the necessary network permissions (iOS/Android differ).
3) In Garmin Pilot, locate the **device/receiver** settings section and select an **external ADS‑B / GDL90 / Stratux-style** source if available.
4) Confirm status indicators show an external source connected and traffic/GPS data present.

### iOS notes

- If prompted, allow **Local Network** access for Garmin Pilot.
- If the connection never comes up, verify it’s enabled in iOS system settings for the app (iOS exposes a per-app Local Network toggle).
- If the app offers an **IP address / host** field for an external receiver:
	- Prefer unicast on home Wi‑Fi (set `gdl90.dest` to the iPhone/iPad IP).
	- Use port `4000`.

### Android notes

- If prompted, allow local network / nearby devices access.
- Some Android builds gate Wi‑Fi discovery behind **Location** permission/services. If Garmin Pilot can’t discover anything, temporarily allow Location (and ensure Location services are enabled) while you set up the connection.
- If the app offers an **IP address / host** field for an external receiver:
	- Prefer unicast on home Wi‑Fi (set `gdl90.dest` to the Android device IP).
	- Use port `4000`.

## Troubleshooting

- Use listen mode to confirm packets are arriving on the expected port.
- On home Wi‑Fi, prefer unicast; some networks block broadcast.
- If Garmin Pilot only supports a specific receiver type on your platform, tell me what choices it presents and I’ll map Stratux-NG to the closest supported option.
