#!/usr/bin/env bash
# Write static/fonts/google-sans-flex.woff2 from the phone's font: the weight axis from 400 to 700,
# every other axis at its default, the Latin ranges only.
set -euo pipefail

cd "$(dirname "$0")/.."
src=../native/android/app/src/main/res/font/google_sans_flex.ttf
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

nix shell nixpkgs#python3Packages.fonttools nixpkgs#woff2 --command sh -c '
	set -eu
	drop=$(fonttools ttx -q -t fvar -o - "$1" | sed -n "s/.*<AxisTag>\(.*\)<\/AxisTag>.*/\1=drop/p" | grep -v "^wght=")
	fonttools varLib.instancer -q "$1" wght=400:700 $drop -o "$2/flex.ttf"
	fonttools subset "$2/flex.ttf" --layout-features="*" \
		--unicodes="U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2190-2199,U+2212,U+2215,U+FEFF,U+FFFD" \
		--output-file="$2/google-sans-flex.ttf"
	woff2_compress "$2/google-sans-flex.ttf" >/dev/null
	cp "$2/google-sans-flex.woff2" "$3"
' sh "$src" "$tmp" static/fonts/google-sans-flex.woff2
cp ../native/android/app/src/main/assets/google-sans-flex-OFL.txt static/fonts/OFL.txt
