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

# Shared edits and the web app

An edit a user can make lives once, in `internal/operations`, addressed by IDs,
and gets an entry in `operations.Actions` marked `TUI`, `Web` or `Both`. The TUI
handlers and `internal/web` (`ordo serve`) both call it; the TUI keeps only its
cursor, modes, undo history and status line around the call. `TestWebCoversItsActions`
goes red when a `Web` action has no handler in `webActions`.

The web app is server-rendered `html/template` with vendored htmx and SortableJS
at pinned versions, one hand-written `app.js`, and a hand-written Material 3
`app.css` whose colors are tokens on `:root`, redefined for dark mode. Its CSP
allows only same-origin scripts and styles, so markup carries no inline script or
`style` attribute. Run `./testdata/web/run.sh [shot-name]` after a UI change: it
seeds `testdata/web/seed.json`, serves it, and writes light and dark PNGs at phone
width to `/tmp/ordo-web/out/` (`ORDO_WEB_SIZE=340x734` for a narrow phone). Read
them, then `rm -rf /tmp/ordo-web`. Add a shot in `shots.mjs` for a new screen or
sheet.
