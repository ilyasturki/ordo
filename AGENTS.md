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

# Shared edits and the app

An edit a user can make lives once, in `internal/operations`, addressed by IDs,
and gets an entry in `operations.Actions` marked `TUI`, `App` or `Both`. The TUI
handlers call it and keep only their cursor, modes, undo history and status line
around the call. An `App` action is an `operations.Op` run by `operations.Apply`:
the app runs it on its own copy through `internal/core`, built to WebAssembly
from `cmd/ordo-core`, and `ordo serve` replays the app's queue through it.
`TestEveryAppActionApplies` goes red when an `App` action has no applier, and
`operations.Undo` gives each op the ops that take it back.

A queued op is replayable: it names where its rows end up, so landing twice or on
a file changed since gives the same result. A step from the current state
(`status.cycle`, `task.shift`, `task.nest`) runs in the app only, and
`operations.Perform` turns it into the replayable op that gets queued.
`docs/sync.md` is the contract of `/api` and its op table: a change to an op's
fields or the sync answer updates it in the same commit, and bumps `web.Protocol`
when its Protocol section says so.

The app in `app/` is SvelteKit built static, with Svelte 5, Tailwind 4, bits-ui,
vaul-svelte, Capacitor 8 for the Android shell, and bun. `ordo serve` embeds its
browser build from `internal/web/spa/`, and the APK ships the same build; a Go
build without it serves `placeholder.html`. Its colors are tokens on `:root` in
`app/src/app.css`, redefined for dark mode. Run `bun run check` and `bun run test`
in `app/` after a change to it; the tests load the real core, so run `just wasm`
first. `just web` fills `internal/web/spa/`, and `just android` installs a debug
APK on the connected phone. A change to `app/bun.lock` needs `just web-deps-hash`,
or `nix build` fails on the stale hash of `node_modules`.

Run `./testdata/web/run.sh [shot-name]` after a UI change: it builds the app,
seeds `testdata/web/seed.json`, serves it and writes light and dark PNGs at phone
width to `/tmp/ordo-web/out/` (`ORDO_WEB_SIZE=340x734` for a narrow phone). Read
them, then `rm -rf /tmp/ordo-web`. Add a shot in `fixture-shots.mjs` for a new
screen or sheet. `just phone` reshoots the README's phone screens from
`assets/phone/` after a change to the list, the task page or the add row.

# Native app

`native/` holds a native Android app (Kotlin, Jetpack Compose, Material 3) on
trial beside the Capacitor one. A page is an activity (`PageActivity`, a
`NavKey`), so the system draws its open, close and predictive back; all motion
stays the platform's default, apart from what follows the finger. Its edits run
`internal/core` bound with gomobile in `native/core`, a module of its own that
`./...` and `nix build` skip, so an op change reaches it at the next
`just native-core` with no Kotlin change. Its sync engine, `sync/SyncEngine.kt`,
ports `app/src/lib/state.svelte.ts`: a change to `docs/sync.md` lands in both.
`nix develop .#native` provides its SDK and NDK. `just native-test` runs its Go
and JVM tests, the JVM ones against the real core through
`native/core/cmd/corepipe` and a real `ordo serve`. `just native-ui <serial>`
runs its Compose tests against a seeded `ordo serve` on `ORDO_UI_PORT` (7466), on
the emulator that serial names: Gradle's connected
tasks install on every attached device, the user's phone included.
`just native` installs the `phone` build, R8-optimized and signed with the
debug key under the debug id `io.github.ilyasturki.ordo.compose.dev`.
