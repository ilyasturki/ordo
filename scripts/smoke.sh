#!/usr/bin/env bash
# Run a built binary through the CLI against a throwaway data directory.
# Usage: scripts/smoke.sh <path to ordo>
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <path to ordo>" >&2
  exit 1
fi

bin="$(realpath "$1")"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
export ORDO_DATA_PATH="$tmp/data" ORDO_CONFIG_PATH="$tmp/config"

# Through a file: grep -q closing a pipe early fails the command under pipefail.
expect() {
  local pattern="$1"
  shift
  "$bin" "$@" > "$tmp/out"
  if ! grep -q -- "$pattern" "$tmp/out"; then
    echo "ordo $*: output lacks '$pattern'" >&2
    exit 1
  fi
}

"$bin" --version
"$bin" --help > /dev/null
"$bin" init > /dev/null
"$bin" category add v1.0.0 > /dev/null
"$bin" task add -C v1.0.0 "Smoke task" > /dev/null
expect 'Smoke task' tasks -C v1.0.0
expect '^## v1.0.0' export -f markdown

for shell in bash zsh fish; do
  expect ordo completion "$shell"
done
