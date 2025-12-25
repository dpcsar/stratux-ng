#!/bin/bash -e

# Ensure this stage has a populated rootfs before sub-stages run.
# pi-gen expects each stage to bootstrap (stage0) or copy_previous (later stages).

if [ ! -d "${ROOTFS_DIR}" ]; then
	copy_previous
fi
