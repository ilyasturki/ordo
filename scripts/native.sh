#!/usr/bin/env bash
# Build the native Android app from the flake's native dev shell: internal/core bound with gomobile, then the APK.
#
# Usage: scripts/native.sh core|apk|install [adb serial]|test|ui <adb serial>
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
    out=$(adb -s "$serial" shell am instrument -w io.github.ilyasturki.ordo.compose.dev.test/androidx.test.runner.AndroidJUnitRunner)
    echo "$out"
    grep -q "^OK (" <<<"$out"
    ;;
*)
    echo "usage: scripts/native.sh core|apk|install [adb serial]|test|ui <adb serial>" >&2
    exit 2
    ;;
esac
