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
URL="http://127.0.0.1:$PORT"

rm -rf "$BASE_DIR" && mkdir -p "$DATA" "$OUT"
go build -o ordo ./cmd/ordo

export ORDO_DATA_PATH="$DATA"
export ORDO_CONFIG_PATH="$BASE_DIR/cfg"
./ordo import testdata/web/seed.json -q
./ordo -p Ordo task attach aaaaaaaa-0000-4000-8000-000000000009 testdata/web/reload.png --name reload -q >/dev/null

./ordo serve --listen "127.0.0.1:$PORT" --base-url "$URL" >"$BASE_DIR/serve.log" 2>&1 &
server=$!
trap 'kill $server 2>/dev/null || true' EXIT
for _ in $(seq 50); do curl -fs -o /dev/null "$URL/login" && break; sleep 0.1; done

token=$(./ordo serve pair | sed -n 's#.*/pair/##p')
cookie=$(curl -fs -o /dev/null -D - -d "token=$token" -d "name=Screenshot" "$URL/pair" |
    sed -n 's/^Set-Cookie: ordo_device=\([^;]*\).*/\1/pi')

node testdata/web/shots.mjs "$URL" "$cookie" "$OUT" "$@"
