# The repo-root VERSION file is the single source of truth for the version,
# shared with the Go build (injected via ldflags below) and flake.nix, which
# stamps the same `v`-prefixed value.
version := trim(`cat VERSION`)
commit := `git rev-parse --short HEAD`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-X github.com/ilyasturki/ordo/internal/version.Version=v" + version + \
    " -X github.com/ilyasturki/ordo/internal/version.Commit=" + commit + \
    " -X github.com/ilyasturki/ordo/internal/version.BuildDate=" + build_date

# Run from source on a fresh copy of your data and config in the cache dir, so a dev build never writes the real files (e.g. `just run version`).
run *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    src=${ORDO_DATA_PATH:-$HOME/.local/share/ordo}
    cfg=${ORDO_CONFIG_PATH:-${XDG_CONFIG_HOME:-$HOME/.config}/ordo}
    work=${XDG_CACHE_HOME:-$HOME/.cache}/ordo-run
    mkdir -p "$work/data" "$work/cfg"
    if [ -d "$src" ]; then
        rsync -aL --delete --delete-excluded --exclude '*.lock' --exclude '*.tmp' --exclude web.json "$src/" "$work/data/"
    fi
    if [ -d "$cfg" ]; then
        rsync -aL --delete "$cfg/" "$work/cfg/"
    fi
    ORDO_DATA_PATH="$work/data" ORDO_CONFIG_PATH="$work/cfg" go run -ldflags "{{ldflags}}" ./cmd/ordo {{ARGS}}

# Run from source on your real data and config.
run-real *ARGS:
    go run -ldflags "{{ldflags}}" ./cmd/ordo {{ARGS}}

# Build the binary.
build:
    go build -ldflags "{{ldflags}}" -o ordo ./cmd/ordo

# Bind internal/core into the Android app's library, native/android/app/libs/ordocore.aar (needs nix: the flake's SDK and NDK).
native-core:
    nix develop .#native --command scripts/native.sh core

# Build the Android app, optimized and signed with the debug key, and install it on the connected device or the one SERIAL names.
native *SERIAL:
    nix develop .#native --command scripts/native.sh install {{SERIAL}}

# Run the Android app's tests: the Go wrapper's, then the JVM ones against a real core and ordo serve.
native-test:
    nix develop .#native --command scripts/native.sh test

# Run the Android app's UI tests on the emulator SERIAL names (they install there, never on the phone).
native-ui SERIAL *ARGS:
    nix develop .#native --command scripts/native.sh ui {{SERIAL}} {{ARGS}}

# Install and open the Android app on the USB phone, signed in to a throwaway seeded server (--reset reseeds it and clears the app).
dev-phone *ARGS:
    ./scripts/dev.sh {{ARGS}}

# Run every test, then the nix build.
test:
    go test ./...
    nix build

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

# Bump version (X.Y.Z or major|minor|patch); updates VERSION, commits, tags.
bump version:
    ./scripts/bump.sh {{version}}
