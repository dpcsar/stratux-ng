# pi-gen image plan (Stratux-NG)

This project will eventually ship as a flashable Raspberry Pi OS image built with **pi-gen**.

Target platform policy:
- Raspberry Pi OS **64-bit (arm64)** on **Pi 3 / 4 / 5**.
- Verify on-device with: `uname -m` → `aarch64`.

The goal is to keep the runtime behavior identical between:
- **Dev mode** (`go run` / `make run`)
- **Appliance mode** (systemd service on a prebuilt SD image)

Stratux-NG is designed so the same YAML config drives both.

## Development workflow (fast iteration)

### 1) Run Stratux-NG locally

- `make run` (uses `CONFIG=./config.yaml` by default)
- Or: `STRATUX_NG_CONFIG=./config.yaml go run ./cmd/stratux-ng`

This exercises the full live pipeline. Without SDRs/GPS/AHRS attached you'll
still get heartbeats and AHRS fallback frames, which is enough to verify the web
UI, config plumbing, and log output.

### 2) Enable real decoders (1090 + 978)

Stratux-NG can supervise decoders itself (recommended).

- Enable `adsb1090.enable: true` and `uat978.enable: true` in your config.
- Confirm your SDR serial tags (e.g. `stx:1090:0`, `stx:978:0`) match `lsusb -v`.

Typical decoder command lines (examples; adjust for your install):
- 1090 (FlightAware `dump1090-fa`): `--device-type rtlsdr --device stx:1090:0 --net --net-stratux-port 30006 --modeac`
- 978 traffic (`dump978-fa`): `--sdr driver=rtlsdr,serial=stx:978:0 --json-port 30978`
- 978 weather / FIS-B uplinks (`dump978-fa`): add `--raw-port 30979`

If you only have **one** RTL-SDR dongle and it is already used for 1090, you cannot also run 978 via RTL/Soapy at the same time.
In that case, use a Stratux UATRadio for 978 and run `dump978-fa` in `--stratuxv3` mode, e.g.:

- `dump978-fa --stratuxv3 /dev/stratux-uatradio --json-port 30978 --raw-port 30979`

Notes for 978 weather (FIS-B):
- Stratux-NG does not “decode products” in the image build; it relays **UAT uplinks** as **GDL90 Uplink (0x07)** frames (Stratux-style).
- You must enable the raw uplink TCP endpoint in config (`uat978.decoder.raw_listen` or `uat978.decoder.raw_addr`) and run `dump978-fa` with `--raw-port`.

For local development without SDR hardware you can still validate parsing/frames via tests:
- `go test ./...`

## Image workflow (pi-gen)

This repo now includes a minimal, in-repo scaffold for image generation under:

- [tools/pi-gen/README.md](../tools/pi-gen/README.md)

It provides:
- a custom pi-gen stage (`stage-stratux-ng/`) that installs Stratux-NG + decoders
- a helper script that clones pi-gen, injects the custom stage, and runs pi-gen in Docker

### Prerequisites (host)

Building images uses pi-gen inside Docker and needs a host that can run privileged containers.

- Linux host (or GitHub Actions runner) with Docker available
- Ability to run privileged Docker containers (pi-gen uses loop devices and mounts)
- Sufficient disk space (pi-gen builds can consume many GB)

Notes:
- On amd64 hosts/runners building an arm64 image, you typically need QEMU/binfmt configured (CI does this via `docker/setup-qemu-action`).
- Many pi-gen setups also require `qemu-user-static` (for `qemu-aarch64-static`) on the host.
- If Docker is installed but not usable as your user, add yourself to the `docker` group or run the build under passwordless sudo.

### Guiding principles

- Put mutable state under `/data`.
- Keep config at `/data/stratux-ng/config.yaml`.
- Run `stratux-ng` under systemd and point it at the persistent config via `STRATUX_NG_CONFIG`.

See persistence guidance: [docs/sd-image-persistence.md](sd-image-persistence.md)

### What the image needs to include

**Binaries**
- `/usr/local/bin/stratux-ng`
- Decoder binaries available in PATH (recommended): `dump1090-fa`, `dump978-fa`

**978 FIS-B weather (recommended)**
- Ensure the image’s default config (or first-boot seeded config) enables the 978 raw uplink endpoint.
- Ensure the decoder command line enables `--raw-port` and the port is reachable from Stratux-NG (typically localhost).

**RTL-SDR access (recommended)**
- Ensure the DVB kernel driver does not claim RTL-SDR dongles (common source of “device busy”):
  - Install a modprobe blacklist file (example content): `blacklist dvb_usb_rtl28xxu`
- Ensure udev permissions allow the service user to access USB/RTL-SDR devices.

**Systemd**
- Install and enable the main service: [configs/systemd/stratux-ng.service.example](../configs/systemd/stratux-ng.service.example)
- Optionally add gpsd ordering drop-in when using gpsd: [configs/systemd/stratux-ng-gpsd.conf.example](../configs/systemd/stratux-ng-gpsd.conf.example)

Optional: run decoders as separate services (instead of Stratux-NG supervising them):
- [configs/systemd/dump1090-fa.service.example](../configs/systemd/dump1090-fa.service.example)
- [configs/systemd/dump978-fa.service.example](../configs/systemd/dump978-fa.service.example)
- [configs/systemd/stratux-ng-decoders.conf.example](../configs/systemd/stratux-ng-decoders.conf.example)

**Persistent data**
- Ensure `/data` exists (either a separate partition, or a directory on rootfs for early prototypes).
- Ensure `/data/stratux-ng/config.yaml` exists on first boot (seed from the repo’s `config.yaml` or an image-specific default).

**Udev rules (recommended)**
- GPS stable symlink: `configs/udev/99-stratux-gps.rules.example` → `/etc/udev/rules.d/99-stratux-gps.rules`

Note: Stratux-NG images intentionally do **not** ship the large set of legacy Stratux peripheral udev rules (e.g. Ping/Pong/UATRadio/SoftRF variants) by default. If you have a niche/older device that still needs custom udev matching or driver binding, add your own rules under `/etc/udev/rules.d/` on the device (or bake them into the image stage under `tools/pi-gen/stage-stratux-ng/.../files/etc/udev/rules.d/`).

If you are using a Stratux UATRadio for 978, there is an example rule in `configs/udev/99-stratux-uatradio.rules.example`.

### Suggested pi-gen structure

Create a separate repo (or sibling folder) that contains pi-gen with a custom stage, for example:

- `stage2+` base Raspberry Pi OS (arm64)
- `stage-stratux-ng/01-packages/00-packages`:
  - `gpsd` + `gpsd-clients` (optional)
  - SDR dependencies:
    - `rtl-sdr` (or at least `librtlsdr0` + udev rules)
    - `soapysdr-tools`, `soapysdr-module-rtlsdr`, `libsoapysdr-dev` (for `dump978-fa` using `--sdr driver=rtlsdr,...`)
    - build deps: `git`, `build-essential`, `cmake`, `pkg-config`, `libusb-1.0-0-dev`, `zlib1g-dev`
    - dump978 deps: `libboost-all-dev`

- `stage-stratux-ng/01-packages/01-run.sh` (recommended):
  - Build and install FlightAware `dump1090` and install it as `dump1090-fa`:
    - `git clone https://github.com/flightaware/dump1090.git`
    - `make -j$(nproc)`
    - `install -m 755 dump1090 /usr/local/bin/dump1090-fa`
  - Build and install FlightAware `dump978` (and provide a `dump978-fa` alias if needed):
    - `git clone https://github.com/flightaware/dump978.git`
    - `make -j$(nproc) dump978-fa`
    - `install -m 755 dump978-fa /usr/local/bin/dump978-fa`

- `stage-stratux-ng/02-files/`:
  - copy `stratux-ng` binary → `/usr/local/bin/stratux-ng`
  - copy `config.yaml` → `/data/stratux-ng/config.yaml` (or `/etc/stratux-ng/config.yaml` then copy-on-first-boot)
  - copy systemd units and enable them
  - copy udev rule examples as real rules

### Build command (using this repo’s helper)

From the repo root:

- `make image`

Overrides (if needed):

- `PIGEN_RELEASE=bookworm make image`
- `PIGEN_ARCH=arm64 make image`

Where to find the output:

- pi-gen writes the `.img` artifacts under `build/pi-gen-src/deploy/`

### Acceptance checklist for the image

On a fresh flashed SD card:
- `systemctl status stratux-ng` is active
- Web UI responds on the configured listen addr
- `/api/status` shows `adsb1090` and `uat978` stream status when enabled
- EFB sees GDL90 traffic
- When 978 raw uplinks are enabled, EFB can receive weather (FIS-B) via GDL90 uplink relay

## Recommendation: keep supervision inside Stratux-NG

You *can* run `dump1090-fa` and `dump978-fa` as separate systemd units, but Stratux-NG already includes a supervisor and stream reconnect.

Keeping decoder supervision inside Stratux-NG means:
- fewer units to manage,
- consistent logging/health snapshots in `/api/status`,
- simpler pi-gen stage (install packages + one service).
