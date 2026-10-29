#!/usr/bin/env bash
# Run the app against a throwaway ordo serve seeded from testdata/web/seed.json, in the browser or on the USB phone.
#
# Usage: scripts/dev.sh web|phone [--reset]
# Data and pairings stay in /tmp/ordo-dev between runs; --reset reseeds them.
set -euo pipefail

target=${1:?usage: scripts/dev.sh web|phone [--reset]}
cd "$(dirname "$0")/.."

work=/tmp/ordo-dev
port=7320
[[ ${2:-} == --reset ]] && rm -rf "$work"

export ORDO_DATA_PATH="$work/data" ORDO_CONFIG_PATH="$work/cfg"
scripts/wasm.sh
go build -o "$work/ordo" ./cmd/ordo
if [[ ! -d $work/data ]]; then
    mkdir -p "$work/data"
    "$work/ordo" import testdata/web/seed.json -q
fi

"$work/ordo" serve --listen "127.0.0.1:$port" >"$work/serve.log" 2>&1 &
server=$!
trap 'kill $server 2>/dev/null || true' EXIT
for _ in $(seq 50); do curl -fs -o /dev/null "http://127.0.0.1:$port/api/hello" && break; sleep 0.1; done

cd app
bun install --frozen-lockfile
case $target in
web)
    echo
    echo "Pair this browser: $("$work/ordo" serve pair --base-url http://localhost:5173 | head -1)"
    echo
    ORDO_SERVE="http://127.0.0.1:$port" bun run dev
    ;;
phone)
    adb reverse "tcp:$port" "tcp:$port"
    bun run android:install
    adb shell monkey -p io.github.ilyasturki.ordo.dev -c android.intent.category.LAUNCHER 1 >/dev/null 2>&1
    echo
    echo "Paste this link on the phone's sign-in screen:"
    "$work/ordo" serve pair --base-url "http://localhost:$port"
    echo
    echo "Serving until Ctrl+C (log: $work/serve.log)"
    wait
    ;;
*)
    echo "usage: scripts/dev.sh web|phone [--reset]" >&2
    exit 2
    ;;
esac
