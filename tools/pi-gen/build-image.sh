#!/usr/bin/env bash
set -euo pipefail

# Build a Stratux-NG Raspberry Pi OS image using pi-gen in Docker.
# This is intentionally minimal and designed to be run from the repo root.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build"
PIGEN_DIR="${BUILD_DIR}/pi-gen-src"
STAGE_SRC_DIR="${ROOT_DIR}/tools/pi-gen/stage-stratux-ng"

PIGEN_REPO_URL="${PIGEN_REPO_URL:-https://github.com/RPi-Distro/pi-gen.git}"
PIGEN_REF="${PIGEN_REF:-2025-12-04-raspios-trixie-arm64}"
PIGEN_RELEASE="${PIGEN_RELEASE:-trixie}"
PIGEN_ARCH="${PIGEN_ARCH:-arm64}"

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

# 3) Copy our custom stage into the pi-gen tree.
# pi-gen supports custom stage names via STAGE_LIST.
STAGE_DST_DIR="${PIGEN_DIR}/stage-stratux-ng"
rm -rf "${STAGE_DST_DIR}"
cp -a "${STAGE_SRC_DIR}" "${STAGE_DST_DIR}"

# 4) Inject the freshly built binary + current repo config into the stage files.
mkdir -p "${STAGE_DST_DIR}/files/usr/local/bin"
install -m 0755 "${BIN_OUT}" "${STAGE_DST_DIR}/files/usr/local/bin/stratux-ng"

mkdir -p "${STAGE_DST_DIR}/files/data/stratux-ng"
install -m 0644 "${ROOT_DIR}/config.yaml" "${STAGE_DST_DIR}/files/data/stratux-ng/config.yaml"

# 5) Write a minimal pi-gen config.
cat >"${PIGEN_DIR}/config" <<EOF
IMG_NAME="stratux-ng"
RELEASE="${PIGEN_RELEASE}"
DEPLOY_COMPRESSION="xz"
ARCH="${PIGEN_ARCH}"
STAGE_LIST="stage0 stage1 stage2 stage-stratux-ng"
FIRST_USER_NAME="pi"
FIRST_USER_PASS="raspberry"
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
