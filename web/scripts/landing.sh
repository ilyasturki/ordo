#!/usr/bin/env bash
# Write the landing's files in static/: the phone's list panel cut from the README's
# assets/phone-light.png and phone-dark.png, the Open Graph card, and Fira Code for the terminal
# mocks (weights 400 to 700, ASCII, Latin-1, arrows and geometric shapes, no ligatures).
# Run it again after a reshoot.
set -euo pipefail

cd "$(dirname "$0")/.."
out=static/landing

nix shell nixpkgs#imagemagick --command sh -c '
	set -eu
	for scheme in light dark; do
		in=../assets/phone-$scheme.png
		third=$(( $(magick identify -format "%w" "$in") / 3 ))
		magick "$in" -crop "${third}x+0+0" +repage -trim +repage -resize 480x -quality 80 "$1/phone-$scheme.webp"
	done
' sh "$out"
cp ../assets/social-preview.png "$out/social-preview.png"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
nix shell nixpkgs#python3Packages.fonttools nixpkgs#woff2 --command sh -c '
	set -eu
	fonttools varLib.instancer -q "$1" wght=400:700 -o "$2/fira.ttf"
	fonttools subset "$2/fira.ttf" --layout-features="" \
		--unicodes="U+0020-007E,U+00A0-00FF,U+2190-2199,U+25A0-25FF" \
		--output-file="$2/fira-code.ttf"
	woff2_compress "$2/fira-code.ttf" >/dev/null
	cp "$2/fira-code.woff2" "$3"
' sh "$(nix build --no-link --print-out-paths nixpkgs#fira-code)/share/fonts/truetype/FiraCode-VF.ttf" "$tmp" static/fonts/fira-code.woff2
{
	echo 'Copyright 2014-2021 The Fira Code Project Authors (https://github.com/tonsky/FiraCode)'
	tail -n +2 static/fonts/OFL.txt
} >static/fonts/fira-code-OFL.txt
