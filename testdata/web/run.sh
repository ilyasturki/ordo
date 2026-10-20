#!/usr/bin/env bash
# Screenshot the web app at phone width, light and dark, against an isolated data dir.
#
# Usage: ./testdata/web/run.sh [shot-name ...]
# PNGs land in /tmp/ordo-web/out/ (shot names are listed in fixture-shots.mjs).
# ORDO_WEB_SIZE=340x734 shoots at another viewport (a Galaxy S21 FE is 340 wide).
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"

BASE_DIR="/tmp/ordo-web"
DATA="$BASE_DIR/data"
OUT="$BASE_DIR/out"
PORT="${ORDO_WEB_PORT:-7391}"

rm -rf "$BASE_DIR" && mkdir -p "$DATA" "$OUT"
go build -o ordo ./cmd/ordo

export ORDO_DATA_PATH="$DATA"
export ORDO_CONFIG_PATH="$BASE_DIR/cfg"
./ordo import testdata/web/seed.json -q
./ordo -p Ordo task attach aaaaaaaa-0000-4000-8000-000000000009 testdata/web/reload.png --name reload -q >/dev/null

testdata/web/shoot.sh ./ordo "$BASE_DIR" "$PORT" "$OUT" "$@"
