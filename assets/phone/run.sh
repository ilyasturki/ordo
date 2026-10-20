#!/usr/bin/env bash
# Shoot the README's phone screens of the demo's ferry project into assets/phone-light.png and assets/phone-dark.png.
#
# Usage: ./assets/phone/run.sh (needs go, node 22+, chromium and nix). ORDO_WEB_PORT picks the port.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export ORDO_DATA_PATH="$work/data" ORDO_CONFIG_PATH="$work/config" TZ=UTC
ordo="$work/ordo"

go build -o "$ordo" ./cmd/ordo
"$ordo" import assets/demo/homelab.json -q
"$ordo" import assets/demo/ferry.json -q
# Seeded here rather than in ferry.json, whose estimates would show in the TUI demo's category headers.
"$ordo" -p ferry task edit t-a1 -e 2d -q --description "Encrypt the values marked secret with age before they are written, so a dotfiles repo is safe to push. One key per host, under ~/.config/ferry/keys."
"$ordo" -p ferry task edit t-a2 -e 4h -q
"$ordo" -p ferry task edit t-a3 -e 1d -q
"$ordo" -p ferry task edit t-n1 -e 2h -q

mkdir "$work/out"
ORDO_WEB_SHOTS=assets/phone/shots.mjs testdata/web/shoot.sh "$ordo" "$work" "${ORDO_WEB_PORT:-7392}" "$work/out" >/dev/null

magick=$(nix build --no-link --print-out-paths nixpkgs#imagemagick)/bin/magick
for scheme in light dark; do
    # The edge keeps a light phone visible on GitHub's white page, and a dark one on its dark page.
    if [ "$scheme" = light ]; then edge="#d0d7de"; else edge="#3d444d"; fi
    frames=()
    for shot in list task add; do
        frame="$work/$shot-$scheme.png"
        "$magick" "$work/out/${shot}_$scheme.png" -resize 560x \
            \( +clone -alpha extract -fill black -colorize 100 -fill white -draw "roundrectangle 0,0,%[fx:w-1],%[fx:h-1],36,36" \) \
            -alpha off -compose CopyOpacity -composite \
            -fill none -stroke "$edge" -strokewidth 2 -draw "roundrectangle 1,1,%[fx:w-2],%[fx:h-2],35,35" "$frame"
        frames+=("$frame")
    done
    "$magick" -background none "${frames[@]}" -splice 32x0 +append -chop 32x0 +repage "assets/phone-$scheme.png"
done

nix run nixpkgs#pngquant -- --quality 80-95 --speed 1 --force --ext .png assets/phone-light.png assets/phone-dark.png
nix run nixpkgs#oxipng -- -q -o4 --strip safe assets/phone-light.png assets/phone-dark.png
