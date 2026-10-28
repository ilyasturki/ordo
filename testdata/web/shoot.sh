#!/usr/bin/env bash
# Serve ORDO_DATA_PATH, pair a device and run shots.mjs against it in Roboto, the font an Android phone shows.
#
# Usage: testdata/web/shoot.sh <ordo-binary> <work-dir> <port> <out-dir> [shot-name ...]
# ORDO_WEB_SHOTS picks the shot list (see shots.mjs). The font goes through fontconfig, so it applies on Linux only.
set -euo pipefail

ordo=$1 work=$2 port=$3 out=$4
shift 4
url="http://127.0.0.1:$port"

roboto=$(nix build --no-link --print-out-paths nixpkgs#roboto)
cat >"$work/fonts.conf" <<CONF
<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <dir>$roboto/share/fonts</dir>
  <include ignore_missing="yes">/etc/fonts/fonts.conf</include>
</fontconfig>
CONF

"$ordo" serve --listen "127.0.0.1:$port" --base-url "$url" >"$work/serve.log" 2>&1 &
server=$!
trap 'kill $server 2>/dev/null || true' EXIT
for _ in $(seq 50); do curl -fs -o /dev/null "$url/api/hello" && break; sleep 0.1; done

token=$("$ordo" serve pair | sed -n 's#.*/pair/##p')
cookie=$(curl -fs -o /dev/null -D - -H 'Content-Type: application/json' \
    -d "{\"token\": \"$token\", \"name\": \"Screenshot\", \"cookie\": true}" "$url/api/pair" |
    sed -n 's/^Set-Cookie: ordo_device=\([^;]*\).*/\1/pi')

FONTCONFIG_FILE="$work/fonts.conf" node "$(dirname "$0")/shots.mjs" "$url" "$cookie" "$out" "$@"
