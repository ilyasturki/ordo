# The repo-root VERSION file is the single source of truth for the version,
# shared with the Go build (injected via ldflags below) and flake.nix, which
# stamps the same `v`-prefixed value.
version := trim(`cat VERSION`)
commit := `git rev-parse --short HEAD`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-X github.com/ilyasturki/phasionary/internal/version.Version=v" + version + \
    " -X github.com/ilyasturki/phasionary/internal/version.Commit=" + commit + \
    " -X github.com/ilyasturki/phasionary/internal/version.BuildDate=" + build_date

# Run from source, passing args through (e.g. `just run version`).
run *ARGS:
    go run -ldflags "{{ldflags}}" ./cmd/phasionary {{ARGS}}

# Build the binary.
build:
    go build -ldflags "{{ldflags}}" -o phasionary ./cmd/phasionary

# Run every test, then the nix build.
test:
    go test ./...
    nix build

# Format Go files and tidy go.mod.
fmt:
    gofmt -w cmd internal
    go mod tidy

# Bump version (X.Y.Z or major|minor|patch); updates VERSION, commits, tags.
bump version:
    ./scripts/bump.sh {{version}}
