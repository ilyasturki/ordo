# The repo-root VERSION file is the single source of truth for the version,
# shared with the Go build (injected via ldflags below) and flake.nix, which
# stamps the same `v`-prefixed value.
version := trim(`cat VERSION`)
commit := `git rev-parse --short HEAD`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-X github.com/ilyasturki/ordo/internal/version.Version=v" + version + \
    " -X github.com/ilyasturki/ordo/internal/version.Commit=" + commit + \
    " -X github.com/ilyasturki/ordo/internal/version.BuildDate=" + build_date

# Run from source, passing args through (e.g. `just run version`).
run *ARGS:
    go run -ldflags "{{ldflags}}" ./cmd/ordo {{ARGS}}

# Build the binary, with the app bundle `ordo serve` embeds.
build: web
    go build -ldflags "{{ldflags}}" -o ordo ./cmd/ordo

# Build internal/core to WebAssembly in app/static, beside the wasm_exec.js of the same Go.
wasm:
    ./scripts/wasm.sh

# Build the browser app into internal/web/spa, the bundle `ordo serve` embeds (needs bun).
web:
    ./scripts/web.sh

# Install a debug build of the Android app on the connected device (needs bun, a JDK and the Android SDK).
android: wasm
    cd app && bun install --frozen-lockfile && bun run android:install

# Bind internal/core into the native Android app's library, native/android/app/libs/ordocore.aar (needs nix: the flake's SDK and NDK).
native-core:
    nix develop .#native --command scripts/native.sh core

# Build the native Android app, optimized and signed with the debug key, and install it on the connected device or the one SERIAL names.
native *SERIAL:
    nix develop .#native --command scripts/native.sh install {{SERIAL}}

# Run the native app's tests: the Go wrapper's, then the JVM ones against a real core and ordo serve.
native-test:
    nix develop .#native --command scripts/native.sh test

# Run the native app's UI tests on the emulator SERIAL names (they install there, never on the phone).
native-ui SERIAL:
    nix develop .#native --command scripts/native.sh ui {{SERIAL}}

# Run the app with hot reload in the browser, against a throwaway seeded server (--reset reseeds it).
dev *ARGS:
    ./scripts/dev.sh web {{ARGS}}

# Install and open the app on the USB phone, against a throwaway seeded server (--reset reseeds it).
dev-phone *ARGS:
    ./scripts/dev.sh phone {{ARGS}}

# Run every test, then the nix build.
test:
    go test ./...
    nix build

# Recompute this system's hash of the app's node_modules in nix/web-deps-hashes.nix, after a change to app/bun.lock.
web-deps-hash:
    ./nix/refresh-web-deps-hash.sh

# Format Go files and tidy go.mod.
fmt:
    gofmt -w cmd internal
    go mod tidy

# Record the README demo GIFs and the social preview (needs go, vhs, nix and IosevkaTerm Nerd Font).
demo:
    vhs assets/demo-dark.tape
    vhs assets/demo-light.tape
    vhs assets/social-preview.tape
    nix run nixpkgs#gifsicle -- --batch -O3 --lossy=20 assets/demo-dark.gif assets/demo-light.gif
    ls -lh assets/demo-dark.gif assets/demo-light.gif assets/social-preview.png

# Shoot the README's phone screenshots (needs go, bun, node 22+, chromium and nix).
phone:
    ./assets/phone/run.sh
    ls -lh assets/phone-light.png assets/phone-dark.png

# Bump version (X.Y.Z or major|minor|patch); updates VERSION, commits, tags.
bump version:
    ./scripts/bump.sh {{version}}
