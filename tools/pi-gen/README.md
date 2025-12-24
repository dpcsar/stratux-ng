# pi-gen (Stratux-NG image generation)

This folder contains a minimal **pi-gen custom stage** that turns a base Raspberry Pi OS image into a Stratux-NG appliance image.

What it does:
- Installs runtime dependencies (Wi‑Fi AP tooling, gpsd optional deps, SDR libs)
- Builds/installs FlightAware decoders (`dump1090-fa`, `dump978-fa`)
- Installs `stratux-ng` + a systemd unit
- Seeds `/data/stratux-ng/config.yaml` (for first boot; persistence is handled separately)

## Prerequisites (host)

- Linux host with `docker` and `git`
- Go toolchain (for building the `stratux-ng` arm64 binary locally)

## Quick start

From the repo root:

- `make image`

Or directly:

- `./tools/pi-gen/build-image.sh`

### Outputs

pi-gen writes images under the pi-gen checkout’s `deploy/` directory (inside `build/pi-gen-src/` by default).

## Notes

- Default `ARCH` is `arm64`.
- Default `RELEASE` is `trixie` because this repo targets Raspberry Pi OS trixie in docs; override if needed:
  - `PIGEN_RELEASE=bookworm make image`
- The stage seeds config at `/data/stratux-ng/config.yaml`. For real appliance images, you typically mount a persistent partition at `/data` (see [docs/sd-image-persistence.md](../../docs/sd-image-persistence.md)).
