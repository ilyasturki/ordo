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
the Android app runs it on its own copy through `internal/core`, bound with
gomobile in `native/core`, and `ordo serve` replays the app's queue through it.
`TestEveryAppActionApplies` goes red when an `App` action has no applier, and
`operations.Undo` gives each op the ops that take it back.

A queued op is replayable: it names where its rows end up, so landing twice or on
a file changed since gives the same result. A step from the current state
(`status.cycle`, `task.shift`, `task.nest`) runs in the app only, and
`operations.Perform` turns it into the replayable op that gets queued.
`docs/sync.md` is the contract of `/api` and its op table: a change to an op's
fields or the sync answer updates it and `sync/SyncEngine.kt` in the same commit,
and bumps `web.Protocol` when its Protocol section says so.

# The app

`native/` holds the Android app (Kotlin, Jetpack Compose, Material 3). A page is
an activity (`PageActivity`, a `NavKey`), so the system draws its open, close and
predictive back; all motion stays the platform's default, apart from what follows
the finger. Behaviour comes from the TUI; only the feel comes from Google Tasks.
`native/core` is a module of its own that `./...` and `nix build` skip, so an op
change reaches the app at the next `just native-core`, with no Kotlin change.
`nix develop .#native` provides its SDK and NDK; `release.yml`'s `apk` job and
`ci.yml`'s `native` job name the same versions as `nix/android-sdk.nix`.

The app id is `io.github.ilyasturki.ordo`, and `.dev` for the `debug` and `phone`
builds, while the Kotlin packages stay `io.github.ilyasturki.ordo.compose`.
`src/debug` also feeds the `phone` build, and holds what a release must lack: the
"Ordo Dev" label and `DevSignInActivity`, the adb way in. A local `assembleRelease`
is unsigned unless `ORDO_KEYSTORE` and its three companions are set.

Run after a change to the app:

- `just native-test`: its Go and JVM tests, the JVM ones against the real core
  through `native/core/cmd/corepipe` and a real `ordo serve`.
- `just native-ui <serial>`: its Compose tests, starting from `AppFixture`, against
  a seeded `ordo serve` on `ORDO_UI_PORT` (7466), on the emulator that serial names:
  Gradle's connected tasks install on every attached device, the user's phone
  included.
- Check screens by eye, light and dark, on an emulator: `ANDROID_SERIAL=<serial>
  just dev-phone` installs the `phone` build signed in to a server seeded from
  `testdata/native/seed.json`, then `adb -s <serial> exec-out screencap -p`, with
  `adb -s <serial> shell cmd uimode night yes` for dark. The README's
  `assets/phone-light.png` and `phone-dark.png` are such shots of the list, a task
  and the add sheet on `assets/demo/ferry.json`: retake them after a change to one.

`just native` installs the `phone` build, R8-optimized and signed with the debug
key, on the connected phone.
