#!/usr/bin/env bash
set -euo pipefail

# Build a Stratux-NG Raspberry Pi OS image using pi-gen in Docker.
# This is intentionally minimal and designed to be run from the repo root.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build"
PIGEN_DIR="${BUILD_DIR}/pi-gen-src"
STAGE_SRC_DIR="${ROOT_DIR}/tools/pi-gen/stage-stratux-ng"

PIGEN_REPO_URL="${PIGEN_REPO_URL:-https://github.com/RPi-Distro/pi-gen.git}"
PIGEN_REF="${PIGEN_REF:-arm64}"
PIGEN_RELEASE="${PIGEN_RELEASE:-trixie}"
PIGEN_ARCH="${PIGEN_ARCH:-arm64}"

# FlightAware decoder sources (overridable via environment variables).
FLIGHTAWARE_DUMP1090_REPO="${FLIGHTAWARE_DUMP1090_REPO:-https://github.com/flightaware/dump1090.git}"
FLIGHTAWARE_DUMP1090_REF="${FLIGHTAWARE_DUMP1090_REF:-v10.2}"
FLIGHTAWARE_DUMP978_REPO="${FLIGHTAWARE_DUMP978_REPO:-https://github.com/flightaware/dump978.git}"
FLIGHTAWARE_DUMP978_REF="${FLIGHTAWARE_DUMP978_REF:-v10.2}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: missing required command: $1" >&2
    exit 1
  }
}

require_cmd git
require_cmd docker
require_cmd go

mkdir -p "${BUILD_DIR}"

# 1) Build the Stratux-NG binary for arm64.
BIN_OUT="${BUILD_DIR}/stratux-ng-linux-arm64"
(
  cd "${ROOT_DIR}"
  echo "==> building stratux-ng (linux/arm64)"
  CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "${BIN_OUT}" ./cmd/stratux-ng
)

# 2) Clone pi-gen (or update it).
if [[ ! -d "${PIGEN_DIR}/.git" ]]; then
  echo "==> cloning pi-gen into ${PIGEN_DIR}"
  git clone "${PIGEN_REPO_URL}" "${PIGEN_DIR}"
fi

(
  cd "${PIGEN_DIR}"
  echo "==> updating pi-gen"
  git fetch --all --tags --prune
  if [[ -n "${PIGEN_REF}" ]]; then
    git checkout -f "${PIGEN_REF}"
  else
    git checkout -f master || true
  fi
)

# Disable exporting the default stage2 image; we want the exported image to come from
# our custom stage (stage-stratux-ng) so it includes the injected binaries/config.
if [[ -d "${PIGEN_DIR}/stage2" ]]; then
  : >"${PIGEN_DIR}/stage2/SKIP_IMAGES"
fi

# 2b) Best-effort: ensure loop module present on the host.
# (The build runs in a privileged container, but loop devices come from the host kernel.)
sudo -n modprobe loop >/dev/null 2>&1 || true

# 2c) Work around util-linux/losetup occasionally returning "/dev/loopN (lost)".
# pi-gen's helper previously parsed the loop number via sed on the raw output; when
# "(lost)" is present it can invoke mknod with an invalid minor and export-image fails.
(
  cd "${PIGEN_DIR}"
  COMMON_SH="${PIGEN_DIR}/scripts/common"
  if [[ -f "${COMMON_SH}" ]] && grep -q '^ensure_next_loopdev()' "${COMMON_SH}"; then
    if grep -q 'loopdev_raw=' "${COMMON_SH}"; then
      echo "==> pi-gen scripts/common already patched (loopdev workaround)"
    else
      echo "==> patching pi-gen scripts/common (loopdev '(lost)' workaround)"
      perl -0777 -i -pe 's/ensure_next_loopdev\(\) \{.*?\n\}\nexport -f ensure_next_loopdev/ensure_next_loopdev() {\n\tlocal loopdev_raw loopdev loopnum\n\tloopdev_raw="\$(losetup -f)"\n\t# util-linux can return values like: "\/dev\/loop13 (lost)"; sanitize to "\/dev\/loop13".\n\tloopdev="\${loopdev_raw%% *}"\n\tif [[ "\$loopdev" =~ ^\/dev\/loop([0-9]+)\$ ]]; then\n\t\tloopnum="\${BASH_REMATCH[1]}"\n\telse\n\t\techo "ERROR: unexpected losetup -f output: \${loopdev_raw}" >&2\n\t\treturn 1\n\tfi\n\t[[ -b "\$loopdev" ]] || mknod "\$loopdev" b 7 "\$loopnum"\n}\nexport -f ensure_next_loopdev/sms' "${COMMON_SH}"
    fi
  fi
)

# 3) Copy our custom stage into the pi-gen tree.
# pi-gen supports custom stage names via STAGE_LIST.
STAGE_DST_DIR="${PIGEN_DIR}/stage-stratux-ng"
rm -rf "${STAGE_DST_DIR}"
cp -a "${STAGE_SRC_DIR}" "${STAGE_DST_DIR}"

# Ensure stage scripts are executable (some file transfers/checkouts can lose mode bits).
if [[ -f "${STAGE_DST_DIR}/prerun.sh" ]]; then
  chmod +x "${STAGE_DST_DIR}/prerun.sh" || true
fi
find "${STAGE_DST_DIR}" -type f -name '*-run.sh' -exec chmod +x {} + || true

# Record decoder versions for the stage scripts.
cat >"${STAGE_DST_DIR}/00-stratux-ng/flightaware-versions.env" <<EOF
FLIGHTAWARE_DUMP1090_REPO=${FLIGHTAWARE_DUMP1090_REPO@Q}
FLIGHTAWARE_DUMP1090_REF=${FLIGHTAWARE_DUMP1090_REF@Q}
FLIGHTAWARE_DUMP978_REPO=${FLIGHTAWARE_DUMP978_REPO@Q}
FLIGHTAWARE_DUMP978_REF=${FLIGHTAWARE_DUMP978_REF@Q}
EOF

# 4) Inject the freshly built binary + current repo config into the stage files.
mkdir -p "${STAGE_DST_DIR}/00-stratux-ng/files/usr/local/bin"
install -m 0755 "${BIN_OUT}" "${STAGE_DST_DIR}/00-stratux-ng/files/usr/local/bin/stratux-ng"
mkdir -p "${STAGE_DST_DIR}/00-stratux-ng/files/data/stratux-ng"
install -m 0644 "${ROOT_DIR}/config.yaml" "${STAGE_DST_DIR}/00-stratux-ng/files/data/stratux-ng/config.yaml"

# 5) Write a minimal pi-gen config.
cat >"${PIGEN_DIR}/config" <<EOF
IMG_NAME="stratux-ng"
RELEASE="${PIGEN_RELEASE}"
DEPLOY_COMPRESSION="xz"
LOCALE_DEFAULT="en_US.UTF-8"
TIMEZONE_DEFAULT="UTC"
KEYBOARD_DEFAULT="us"
WPA_COUNTRY="US"
ARCH="${PIGEN_ARCH}"
STAGE_LIST="stage0 stage1 stage2 stage-stratux-ng"
FIRST_USER_NAME="pi"
FIRST_USER_PASS="pi"
DISABLE_FIRST_BOOT_USER_RENAME="1"
ENABLE_SSH="1"
EOF

# 6) Run pi-gen in Docker.
(
  cd "${PIGEN_DIR}"
  echo "==> running pi-gen (docker)"
  # pi-gen’s recommended Docker entrypoint is build-docker.sh.
  # If your pi-gen checkout does not have it, run pi-gen per its upstream docs.
  if [[ -x ./build-docker.sh ]]; then
    ./build-docker.sh
  else
    echo "error: ./build-docker.sh not found in pi-gen checkout" >&2
    echo "hint: check pi-gen upstream docs; your checkout may be different" >&2
    exit 1
  fi
)

echo "==> done"
echo "Images should be under: ${PIGEN_DIR}/deploy/"
