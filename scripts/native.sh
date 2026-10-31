#!/usr/bin/env bash
# Build the Android app from the flake's native dev shell: internal/core bound with gomobile, then the APK.
#
# Usage: scripts/native.sh core|apk|install [adb serial]|test|ui <adb serial> [am instrument flags]
set -euo pipefail
cd "$(dirname "$0")/.."

: "${ANDROID_HOME:?run it through just, in the native dev shell of the flake}"
root=$PWD
apk=native/android/app/build/outputs/apk/debug/app-debug.apk
phone=native/android/app/build/outputs/apk/phone/app-phone.apk

core() {
    local bin=$root/native/core/.bin
    GOBIN=$bin go -C native/core install tool
    mkdir -p native/android/app/libs
    (cd native/core && PATH=$bin:$PATH gomobile bind -target=android/arm64,android/amd64 -androidapi 26 \
        -javapkg io.github.ilyasturki.ordo.core -ldflags='-s -w' -trimpath \
        -o "$root/native/android/app/libs/ordocore.aar" ./ordocore)
}

case ${1:-} in
core)
    core
    ;;
apk)
    core
    native/android/gradlew -p native/android assembleDebug
    ;;
install)
    core
    native/android/gradlew -p native/android assemblePhone
    adb ${2:+-s "$2"} install -r "$phone"
    ;;
test)
    go -C native/core vet ./...
    go -C native/core test ./...
    core
    native/android/gradlew -p native/android testDebugUnitTest
    ;;
ui)
    # The device's adb, not Gradle's connected tasks: those install on every device attached.
    serial=${2:?usage: scripts/native.sh ui <adb serial of an emulator>}
    core
    native/android/gradlew -p native/android assembleDebug assembleDebugAndroidTest
    adb -s "$serial" install -r "$apk"
    adb -s "$serial" install -r native/android/app/build/outputs/apk/androidTest/debug/app-debug-androidTest.apk
    # ServerTest signs in to a throwaway seeded ordo serve, reached through adb reverse.
    port=${ORDO_UI_PORT:-7466}
    passphrase=throwaway-ui-passphrase
    data=$(mktemp -d)
    go build -o "$data/ordo" ./cmd/ordo
    export ORDO_DATA_PATH=$data/data ORDO_CONFIG_PATH=$data/config
    "$data/ordo" import testdata/native/seed.json -q
    "$data/ordo" serve passphrase <<<"$passphrase"
    "$data/ordo" serve --listen "127.0.0.1:$port" >"$data/serve.log" 2>&1 &
    served=$!
    trap 'kill "$served"; rm -rf "$data"' EXIT
    adb -s "$serial" reverse "tcp:$port" "tcp:$port"
    out=$(adb -s "$serial" shell am instrument -w -e server "http://127.0.0.1:$port" -e passphrase "$passphrase" "${@:3}" io.github.ilyasturki.ordo.dev.test/androidx.test.runner.AndroidJUnitRunner)
    echo "$out"
    grep -q "^OK (" <<<"$out"
    ;;
*)
    echo "usage: scripts/native.sh core|apk|install [adb serial]|test|ui <adb serial>" >&2
    exit 2
    ;;
esac
