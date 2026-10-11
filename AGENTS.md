# Project Overview

Ordo is a terminal-first, single-user project planning tool. Local-only, offline, keyboard-driven. The implementation targets Go with Bubble Tea for TUI. The app should look well on dark and light mode.

# Build Commands

```bash
go build -o ordo ./cmd/ordo
just test                          # `go test ./...`, then `nix build`
go test -v ./internal/domain/...   # Run tests for specific package
```

# Visual TUI Testing

Run `./testdata/vhs/run.sh [tape-name]` to verify UI changes. It rebuilds,
re-seeds an isolated data dir, runs each `.tape`, and writes PNG screenshots
to `/tmp/ordo-vt/vhs-out/`. Read the PNGs, then `rm -rf /tmp/ordo-vt/vhs-out`.
Add a new tape for any new mode, modal, or non-trivial keybinding.

# File format

`docs/file-format.md` is the contract other apps code against. A change to the
JSON shape (`internal/domain/types.go`), the write and lock path
(`internal/data`), or the stale guard updates it in the same commit, and bumps
`ProjectSchemaVersion` when its Schema section says so.
