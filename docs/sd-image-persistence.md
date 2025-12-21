# Prebuilt SD image: persistence + power-loss resilience

This project is intended to be run as an “appliance” (flash SD card, power on, connect EFB).

This doc focuses on **reducing SD-card writes** and **limiting corruption risk** from sudden power loss.

## What Stratux-NG needs to write

Stratux-NG itself is relatively write-light:

- Config file: written via the Web UI (`/api/settings`) using **atomic write + rename**.
- Record/replay logs: only written when `gdl90.record.enable: true`.
- Normal logs: written to stdout/stderr (when run under systemd, this goes to `journald`).

Because config writes are atomic, extra filesystem layers are **not required** just to keep YAML settings safe.

## Recommended layout (common to both approaches)

Use a persistent data mount for anything you want to survive reboots:

- Mount an ext4 partition at `/data`
- Store Stratux-NG writable files under `/data/stratux-ng/`

Suggested paths:

- Config: `/data/stratux-ng/config.yaml`
- Optional record log: `/data/stratux-ng/gdl90-record.log`

Why ext4 (not `/boot`): Raspberry Pi OS typically mounts `/boot` as **vfat**, and rename/flush semantics are less robust than ext4. Stratux-NG’s atomic save pattern is strongest on a journaling filesystem.

## Recommended approach: writable /data (no overlay)

### Summary

- Keep a normal root filesystem, but reduce background writes.
- Put Stratux-NG config and any high-churn output on `/data`.

### Pros

- Simpler than overlay.
- Easier to manage updates.

### Cons

- More SD writes than overlay (apt, journald, logrotate, etc.), unless tuned.

### OS tuning checklist

These are OS-level knobs (not Stratux-NG code changes):

- Disable swap (`dphys-swapfile` on Raspberry Pi OS) if appropriate.
- Set `journald` to reduce or avoid disk writes:
  - `Storage=volatile` (RAM-only), or
  - cap size/retention (e.g., `SystemMaxUse=`).
- Mount `tmpfs` for hot-write paths:
  - `/tmp`
  - optionally `/var/log` (only if your environment supports it)
- Disable noisy timers/services you don’t need (e.g., `apt-daily*`, `man-db.timer`).
- Use `noatime` on ext4 mounts.

## Systemd service example

This repo ships an example unit:

- [configs/systemd/stratux-ng.service.example](../configs/systemd/stratux-ng.service.example)

It hardens the service and only allows writes under `/data/stratux-ng`.

## Minimal “data partition” fstab snippet

This is just a sketch; you’ll need the correct device/UUID.

```fstab
# /data holds Stratux-NG config and optional recordings.
UUID=<your-uuid>  /data  ext4  defaults,noatime  0  2
```

## Stratux-NG config for appliance use

Point the service at the persistent config:

- `stratux-ng --config /data/stratux-ng/config.yaml`

Example config values commonly used for a Pi AP subnet:

- `gdl90.dest: 192.168.10.255:4000`
- `web.enable: true`
- `web.listen: 0.0.0.0:8080`
