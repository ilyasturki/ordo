#!/usr/bin/env bash
# Run the Android app on the USB phone, signed in to a throwaway ordo serve seeded from testdata/native/seed.json.
#
# Usage: scripts/dev.sh [--reset]
# Data and pairings stay in /tmp/ordo-dev between runs; --reset reseeds them.
set -euo pipefail

cd "$(dirname "$0")/.."

work=/tmp/ordo-dev
port=7320
app=io.github.ilyasturki.ordo.dev
[[ ${1:-} == --reset ]] && rm -rf "$work"

export ORDO_DATA_PATH="$work/data" ORDO_CONFIG_PATH="$work/cfg"
mkdir -p "$work"
go build -o "$work/ordo" ./cmd/ordo
if [[ ! -d $work/data ]]; then
    mkdir -p "$work/data"
    "$work/ordo" import testdata/native/seed.json -q
fi

"$work/ordo" serve --listen "127.0.0.1:$port" >"$work/serve.log" 2>&1 &
server=$!
trap 'kill $server 2>/dev/null || true' EXIT
for _ in $(seq 50); do curl -fs -o /dev/null "http://127.0.0.1:$port/api/hello" && break; sleep 0.1; done

nix develop .#native --command scripts/native.sh install
adb reverse "tcp:$port" "tcp:$port"

# One device on this server, however many runs signed it in.
for id in $("$work/ordo" serve devices | awk 'NR > 1 { print $1 }'); do
    "$work/ordo" serve revoke "$id" >/dev/null 2>&1 || true
done
link=$("$work/ordo" serve pair --base-url "http://localhost:$port" | head -1)
adb shell am start -n "$app/io.github.ilyasturki.ordo.compose.DevSignInActivity" --es link "$link" >/dev/null

signed_in() { [[ $("$work/ordo" serve devices | wc -l) -gt 1 ]]; }
for _ in $(seq 100); do signed_in && break; sleep 0.1; done
if signed_in; then
    echo "Signed in to the test server."
else
    echo "The app didn't sign in (adb logcat says why). Paste this link on its sign-in screen:" >&2
    echo "$link" >&2
fi
echo "Serving until Ctrl+C (log: $work/serve.log)"
wait "$server"
