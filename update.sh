#!/usr/bin/env bash
set -ex

make build
sudo systemctl stop stratux-ng.service
sudo cp bin/stratux-ng /usr/local/bin/stratux-ng
sudo systemctl start stratux-ng.service
sudo systemctl status stratux-ng.service
