#!/usr/bin/env bash
# Seed an isolated Ordo data dir for visual testing.
# Usage: ./seed.sh <data-dir> <config-dir>
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "usage: $0 <data-dir> <config-dir>" >&2
    exit 2
fi

DATA="$1"
CFG="$2"
BIN="${BIN:-./ordo}"
FIXTURE="$(dirname "$0")/seed.json"

rm -rf "$DATA" "$CFG"
mkdir -p "$DATA" "$CFG"

export ORDO_DATA_PATH="$DATA"
export ORDO_CONFIG_PATH="$CFG"

"$BIN" import "$FIXTURE" -q
