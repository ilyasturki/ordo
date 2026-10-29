#!/usr/bin/env bash
# Build the native Android app from the flake's native dev shell: internal/core bound with gomobile, then the APK.
#
# Usage: scripts/native.sh core|apk|install [adb serial]|test
set -euo pipefail
cd "$(dirname "$0")/.."

: "${ANDROID_HOME:?run it through just, in the native dev shell of the flake}"
root=$PWD
apk=native/android/app/build/outputs/apk/debug/app-debug.apk

core() {
    local bin=$root/native/core/.bin
    GOBIN=$bin go -C native/core install tool
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
    native/android/gradlew -p native/android assembleDebug
    adb ${2:+-s "$2"} install -r "$apk"
    ;;
test)
    go -C native/core vet ./...
    go -C native/core test ./...
    core
    native/android/gradlew -p native/android testDebugUnitTest
    ;;
*)
    echo "usage: scripts/native.sh core|apk|install [adb serial]|test" >&2
    exit 2
    ;;
esac
