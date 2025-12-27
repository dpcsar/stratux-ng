#!/bin/bash -e

# Build FlightAware dump1090-fa and dump978-fa from configurable sources.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/flightaware-versions.env"

# Defaults if no overrides are provided.
DUMP1090_REPO_DEFAULT="https://github.com/flightaware/dump1090.git"
DUMP1090_REF_DEFAULT="v10.2"
DUMP978_REPO_DEFAULT="https://github.com/flightaware/dump978.git"
DUMP978_REF_DEFAULT="v10.2"

if [[ -f "${CONFIG_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${CONFIG_FILE}"
fi

DUMP1090_REPO="${FLIGHTAWARE_DUMP1090_REPO:-${DUMP1090_REPO_DEFAULT}}"
DUMP1090_REF="${FLIGHTAWARE_DUMP1090_REF:-${DUMP1090_REF_DEFAULT}}"
DUMP978_REPO="${FLIGHTAWARE_DUMP978_REPO:-${DUMP978_REPO_DEFAULT}}"
DUMP978_REF="${FLIGHTAWARE_DUMP978_REF:-${DUMP978_REF_DEFAULT}}"

ENV_FILE="${ROOTFS_DIR}/tmp/flightaware-env"
cat >"${ENV_FILE}" <<EOF
FLIGHTAWARE_DUMP1090_REPO=${DUMP1090_REPO@Q}
FLIGHTAWARE_DUMP1090_REF=${DUMP1090_REF@Q}
FLIGHTAWARE_DUMP978_REPO=${DUMP978_REPO@Q}
FLIGHTAWARE_DUMP978_REF=${DUMP978_REF@Q}
EOF

on_chroot <<'EOF'
set -euo pipefail

. /tmp/flightaware-env

WORKDIR=/tmp/flightaware-build
rm -rf "${WORKDIR}"
mkdir -p "${WORKDIR}"
cd "${WORKDIR}"

log() {
  echo "[flightaware-build] $1"
}

clone_repo() {
  local repo_url="$1"
  local repo_ref="$2"
  local repo_dir="$3"
  log "cloning ${repo_dir} (${repo_ref})"
  git clone --depth 1 --branch "${repo_ref}" "${repo_url}" "${repo_dir}"
}

build_dump1090() {
  clone_repo "${FLIGHTAWARE_DUMP1090_REPO}" "${FLIGHTAWARE_DUMP1090_REF}" dump1090
  cd dump1090
  log "building dump1090"
  make -j"$(nproc)"
  install -m 0755 dump1090 /usr/local/bin/dump1090-fa
  if [[ -f view1090 ]]; then
    install -m 0755 view1090 /usr/local/bin/view1090-fa
  fi
  if command -v strip >/dev/null 2>&1; then
    strip /usr/local/bin/dump1090-fa || true
    if [[ -f /usr/local/bin/view1090-fa ]]; then
      strip /usr/local/bin/view1090-fa || true
    fi
  fi
  cd ..
}

build_dump978() {
  clone_repo "${FLIGHTAWARE_DUMP978_REPO}" "${FLIGHTAWARE_DUMP978_REF}" dump978
  cd dump978
  log "building dump978"
  make -j"$(nproc)" dump978-fa
  install -m 0755 dump978-fa /usr/local/bin/dump978-fa
  if [[ -f skyaware978 ]]; then
    install -m 0755 skyaware978 /usr/local/bin/skyaware978
  fi
  if command -v strip >/dev/null 2>&1; then
    strip /usr/local/bin/dump978-fa || true
    if [[ -f /usr/local/bin/skyaware978 ]]; then
      strip /usr/local/bin/skyaware978 || true
    fi
  fi
  cd ..
}

build_dump1090
build_dump978

cd /
rm -rf "${WORKDIR}"
EOF

rm -f "${ENV_FILE}"
