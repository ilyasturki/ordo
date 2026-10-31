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

# A 1x1 png.
base64 -d > "$tmp/shot.png" <<< 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=='
"$bin" task attach "Smoke task" "$tmp/shot.png" > /dev/null
"$bin" task attach "Smoke task" - --name "from stdin" < "$tmp/shot.png" > /dev/null
store="$("$bin" attachments path)"
expect "^  shot.png  *$store/[0-9a-f]\{32\}\.png\$" task show "Smoke task"
expect "^    !\[from stdin\]($store/[0-9a-f]\{32\}\.png)\$" export -f markdown
ls "$store"/*.png > /dev/null

for shell in bash zsh fish; do
  expect ordo completion "$shell"
done

"$bin" serve --listen "unix:$tmp/s.sock" > /dev/null 2>&1 &
served=$!
trap 'kill "$served" 2> /dev/null || true; rm -rf "$tmp"' EXIT
serve_get() {
  curl -fsS --unix-socket "$tmp/s.sock" "http://ordo$1"
}
for _ in $(seq 50); do
  serve_get /api/hello > "$tmp/out" 2> /dev/null && break
  sleep 0.1
done
grep -q '"protocol"' "$tmp/out"
serve_get / > "$tmp/out"
grep -q "sync API" "$tmp/out"
expect '/pair/' serve pair --base-url https://ordo.example.com
serve_get "/pair/$(sed -n 's|.*/pair/||p' "$tmp/out")" > "$tmp/out"
grep -q "Ordo app" "$tmp/out"
