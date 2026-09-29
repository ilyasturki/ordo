# Project Overview

Phasionary is a terminal-first, single-user project planning tool. Local-only, offline by default, keyboard-driven. The implementation targets Go with Bubble Tea for TUI. The app should look well on dark and light mode.

# Build Commands

```bash
go build -o phasionary ./cmd/phasionary
just test                          # `go test ./...`, then both nix packages
go test -v ./internal/domain/...   # Run tests for specific package
```

# Visual TUI Testing

Run `./testdata/vhs/run.sh [tape-name]` to verify UI changes. It rebuilds,
re-seeds an isolated data dir, runs each `.tape`, and writes PNG screenshots
to `/tmp/phas-vt/vhs-out/`. Read the PNGs, then `rm -rf /tmp/phas-vt/vhs-out`.
Add a new tape for any new mode, modal, or non-trivial keybinding.

# Web app

`web/` is the Svelte 5 client `phasionary-server` embeds (`internal/webui`).
`just web` is the dev loop. For a fast loop without nix, `npm --prefix web run
check` type-checks and `npm --prefix web test` runs the unit tests. Ops the
client emits are pinned by a fixture (`npm --prefix web run fixtures`) that a
Go test in `internal/server` replays through the real merge — regenerate it
whenever `web/src/lib/ops.ts` changes.
