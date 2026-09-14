# Sync architecture (A2): readable files locally, sync as an app feature

Status: **accepted** — decided 2026-08-29. This document is the in-tree record
of the decision and the design the mobile/web slice builds on.

## The decision

Phasionary gains mobile and web clients backed by an **optional self-hosted
server**, while the terminal app stays purely local. Chosen architecture, out
of four candidates:

**A2 — JSON files locally, app-level sync, SQLite only in the server.**

- Every client owns a full local store and works with zero connectivity.
  The TUI/CLI store stays what it is today: readable JSON files.
- Sync is a feature of the app, not of the filesystem: each device keeps a
  small **change journal** and exchanges journal entries with the server over
  HTTP when one is configured and reachable.
- The server is a single Go binary on the user's own machine. It holds its
  state in SQLite, merges changes per-item, serves the JSON API for the phone,
  and serves the web app itself.
- No server configured → the app is exactly the local, single-device tool it
  is today. Nothing degrades.

### Rejected alternatives, and why

| Option | Shape | Why not |
| --- | --- | --- |
| A1 — SQLite everywhere | Same sync design, binary local store | Gives up greppable, hand-editable, git-diffable data files for robustness this data size doesn't need. A2 → A1 stays a contained swap behind `data.ProjectRepository` if scale ever demands it. |
| B — merge-safe event-log files | Per-device append-only logs, any file syncer is the pipe | The most engineering on the table (event engine, compaction, causal ordering, snapshots — on every platform) to serve users who run a file syncer but won't run one small server. The server must exist anyway for phone/web. |
| C — CRDT storage (Automerge / cr-sqlite) | Library-merged replicas over any transport | Heavyweight young dependency in every client (Rust bindings in Go, WASM on web), binary library-versioned on-disk format. Overkill when "conflict" means one user racing themself. |
| Server as source of truth | All clients online-first | Violates the fixed rule below. |
| Git as the pipe | Commit/merge on every save | Unusable from a phone. |
| Google Tasks / CalDAV backend | Sync into an existing ecosystem | Phases, categories, separators, priorities and estimates don't map; the domain would be flattened into someone else's model. |
| Hosted sync (Turso, PowerSync, Firebase…) | Third-party cloud | Against the self-hosted, offline-by-default ethos. |

## Fixed rules (hold in every future slice)

1. **The TUI and CLI are local-only first.** No network, no daemon, no server
   required, ever. A server is an addition, never a dependency.
2. **Quick capture works offline everywhere.** A phone with no signal still
   captures; entries queue and sync later.
3. **Single user, self-hosted.** No accounts, no multi-tenancy. Server auth is
   a bearer credential per device.
4. **One pipe per data directory.** When app-level sync is enabled, file-level
   syncers (Syncthing, Dropbox, …) must stop carrying the data directory —
   they demote to backup. Two pipes moving the same data would fight; the app
   should detect tell-tale syncer droppings (`*.sync-conflict-*` files) and
   warn.

## Current on-disk layout (schema v1)

What exists after this slice, all under `~/.local/share/phasionary/` (or
`PHASIONARY_DATA_PATH`):

```
projects/<project-id>.json    one project per file, schema-versioned
projects/<id>.json.lock       flock files (advisory, content ignored)
projects/.global.lock         serializes cross-project operations
state.json                    UI state: folds, cursors, project order, dir links
```

Config is separate, `~/.config/phasionary/config.json` (or
`PHASIONARY_CONFIG_PATH`), mode 0600, per-device preferences only.

Properties already in place that sync builds on:

- **Stable random IDs** on projects, categories and tasks (`domain.NewID`),
  never reused, safe as filenames (`domain.ValidateID`).
- **Atomic, fsync'd writes** (`writeFileAtomic`): readers see the old file or
  the new one, never a torn one.
- **Advisory flocks** serialize concurrent local writers (TUI + CLI).
- **Schema versioning** (added this slice): every save stamps
  `"schema": <domain.ProjectSchemaVersion>`; a missing field reads as v1; a
  file declaring a *newer* schema is refused with an "upgrade phasionary"
  error rather than rewritten through an older format — that is what protects
  a half-upgraded fleet.
- **Write-boundary text validation** rejects control characters once, for
  every writer (`domain.ValidateLine` / `ValidateMultiline`).

## Sync design (implemented in later slices)

### Device identity (implemented, slice 2)

Each device that syncs gets a random, immutable device ID, minted at
enrollment (`phasionary sync login <url>`). It lives in
**`$XDG_STATE_HOME/phasionary/device.json`** (`~/.local/state/phasionary/`,
override `PHASIONARY_STATE_PATH`), deliberately **not** in the data or config
directories: both of those are historically carried by file syncers, and a
device ID that syncs to another machine is two devices claiming one identity.
Copying a data directory to a new machine is therefore safe by construction —
the new machine has no state dir, so it enrolls as a new device.

The sync credential (server URL + per-device bearer token, granted at
enrollment) lives in the same 0600 file, in a 0700 state dir.

### Change journal (implemented, slice 2)

A per-device, purely local outbox — bookkeeping, not history
(`internal/journal`). Also in `$XDG_STATE_HOME/phasionary/`
(`journal.jsonl` + `journal.lock` + `journal.head`), never in the synced
data dir.

- **Ops are derived by diffing at the save boundary**, not by instrumenting
  every mutation: one hook (`data.ChangeRecorder`) covers every writer — the
  TUI, the CLI, import.
- Each op is one JSONL line: `journal.Op`, kinds in the `Kind*` constants.
  Update ops carry only changed fields, with cleared values present as
  explicit empties.
- **Order is a field of the container**: whenever a list's ID sequence
  changes, a single `*.reorder` op carries the full new sequence; the server
  never infers order from creates.
- `seq` is a per-device monotonic counter, and never regresses across a
  prune: `journal.head` records the acked high-water mark.
  Cross-device ordering uses the server's arrival order plus per-field
  last-writer-wins with the op timestamp as tiebreaker (single user racing
  themself: LWW per field is the right amount of machinery).
- **Deletes are tombstone ops.** State files just remove the entity; the
  journal entry is what tells the server "deleted", so a delete can never be
  resurrected by a stale peer pushing an old snapshot. A whole-project delete
  is one tombstone; the server cascades it to its children.
- Entries are **pruned after server acknowledgment** (`PruneThrough`). The
  recorder checks for `device.json` on every write rather than once at
  startup, so a TUI that was open when `sync login` ran journals from its
  next save on; while unenrolled it journals nothing and the journal never
  grows. Enrolling starts from a full snapshot upload (exactly the
  diff-from-nothing cascade).

### Protocol (implemented, slice 3)

Wire types live in `internal/syncproto`; the client side in
`internal/syncclient`.

- `POST /v1/enroll`: the device mints its own ID, the server mints the bearer
  token and stores only its hash.
- `POST /v1/sync`: the journal up, snapshots down.

Push and pull happen in one round-trip, and **the server is the only place
merges happen**. A device never applies ops: the response carries a full
snapshot of every project whose server version moved past the device's
cursor — its own push included, so a device sees its edits merged with
everyone else's in the same round — and the device overwrites its files with
those snapshots. Applying ops client-side would not converge on a same-field
conflict without per-field timestamps in the JSON files; snapshots converge
by construction.

Two rules make a round safe to interrupt (`syncclient.Run`): a snapshot is
skipped while the journal still holds entries for that project, and the
device cursor advances only after every snapshot has been applied.

### Stale saves (implemented, slice 3)

The TUI edits in memory and saves whole snapshots, so a project changed on
disk by a pull (or by the CLI) would be silently overwritten by its next
save — and with sync on, the overwrite would journal the reversal. The store
therefore refuses a snapshot save over a file version this process did not
load. Read-modify-write under the flock (`WithProjectLocked`) is never stale;
listing projects deliberately does not move the baseline.

### Server (implemented, slice 3)

`phasionary-server`, a separate binary (`cmd/phasionary-server`,
`internal/server`) so the local-only TUI never links a server or a SQLite
driver. Pure-Go SQLite (`modernc.org/sqlite`) keeps the arm64 release build
toolchain-free.

- **State model:** one `entities` table; every project, category and task is
  a row of merged fields keyed by the domain JSON tags plus a timestamp per
  field. Order lists and a task's `category_id` are fields like any other,
  so every op kind is the same operation: set fields, or tombstone.
- **Merge:** per field, last-writer-wins on the op timestamp, later arrival
  winning a tie. A tombstone, once set, wins over anything after it. Creates
  are upserts, so enrolling with a copy of an already-synced data directory
  converges. An order list that lost the race drops the other side's
  additions from the sequence, not from existence: the snapshot appends
  every live entity the winning list misses, by `created_at`. A task whose
  category was deleted goes with it.
- **Pull:** each project row carries the server cursor at the last request
  that touched it; a pull returns projects with version > the device cursor.
- **Auth:** per-device bearer token minted at enrollment, stored hashed.
  Enrollment codes are minted by `phasionary-server enroll` into the same
  database (WAL, so it runs beside the service), single use, ten-minute
  life; a failed attempt costs a second. Retries after a lost response are
  idempotent through the per-device acked seq.
- Plain HTTP; deployment guidance is private network or TLS reverse proxy.
  The NixOS module (`nix/module.nix`, `services.phasionary-server`) runs it
  hardened under its own user. It also serves the web app below.

### Web app (implemented, slice 4)

Svelte 5 + Vite + TypeScript in `web/`, built by `buildNpmPackage` and
embedded into `phasionary-server` through `internal/webui` — one binary to
deploy. The committed embed directory holds only a placeholder, so `go build`
and `go test ./...` need no Node; a bare binary answers 503 on the app's paths.

- **Local model:** IndexedDB holds whole project snapshots, an append-only
  outbox keyed by the seq the protocol acks, and the device record (id, token,
  cursor, last sync). Every edit writes the project and its ops in one
  transaction, so a reload never shows an edit whose op was lost.
- **Ops:** emitted at the action, not diffed from a before/after tree — each
  action knows what it changed. `web/src/lib/ops.ts` is the whole vocabulary,
  and `web/src/lib/fixtures.test.ts` writes a scripted session's ops to
  `internal/server/testdata/web-ops.json`, which a Go test replays through the
  real merge and compares against the model the client ended up with. A
  renamed field or a dropped reorder fails that test.
- **Sync:** the same round as the Go client — push, prune what was acked,
  then apply snapshots, skipping any project the outbox has touched since the
  push. Automatic on load, after each edit (debounced), on foreground, and on
  `online`; plus a manual button on the sync screen. Snapshots date a project
  by the newest field timestamp or tombstone among its entities, so the list's
  age column tracks renames and deletes.
- **Serving:** hashed assets are immutable, everything else revalidates
  (`sw.js` above all, or a worker would never hand over). Unknown paths fall
  back to the app, except under `/assets/` and `/v1/`, which answer 404 as
  themselves. CSP pins every fetch directive to `'self'` and denies the rest.
- **Host allowlist:** `--allowed-host` / `PHASIONARY_SERVER_ALLOWED_HOSTS`,
  the same DNS-rebinding defence the removed `phasionary serve` carried: IP
  literals and `localhost` always pass, a name must be listed.
- **Offline:** the service worker precaches the shell, so the app opens with
  no network and edits queue. It needs a secure context, so a plain-HTTP
  deployment loses the offline shell and the install prompt but nothing else.

### Setup without ceremony (implemented, slice 5)

The server exists for the web app and for other devices, so joining one had to
stop being a six-step copy of codes between terminals.

- **The host enrolls itself.** On startup the server spends a code on its own
  TUI over loopback — the machine holding the database has nothing to prove to
  the network. It skips a device that is already enrolled (which may name
  another server) and a machine with no data directory. Because the
  enrollment outlives the server process, every start on an enrolled host
  prints that its TUI now syncs and how to log out. `--no-local-enroll`
  turns it off, and the NixOS unit sets it.
- **The phone scans.** The server prints a pairing block on startup when
  stdout is a terminal (never into a journal, where a code would outlive its
  ten minutes), and `phasionary-server pair` prints another. Both show the code as a QR of
  `http://<lan-ip>:<port>/#code=…`, painted with half blocks and explicit
  colours so it reads on any terminal theme. The code rides in the fragment,
  which no proxy or log ever sees; the app lifts it out of `location.hash`,
  clears the URL, and enrolls itself.
- **The address is printed, not guessed.** The default bind is `0.0.0.0`,
  since a phone cannot reach loopback, and `pair` names the interfaces the
  server answers on, LAN before CGNAT (Tailscale).
- **The TUI syncs at its edges.** Pull at launch, push after the saver's final
  flush (`internal/app/autosync.go`); three seconds each, never fatal.

## Slice plan from here

1. **(done)** Android app, `internal/api`, `phasionary serve`,
   serve token and NixOS serve module removed; project files schema-versioned;
   this design recorded.
2. **(done)** **Journal + device identity** in the Go core
   (`internal/journal`, `data.ChangeRecorder`, `sync status`).
3. **(done)** **Server**, client sync commands, the stale-save guard, NixOS
   module.
4. **(done)** **Web app** served by the server: offline-first Svelte client,
   static serving behind a CSP, Host-header allowlist, op-conformance fixture.
5. **(done)** **Setup without ceremony**: the host enrolls itself, `pair`
   prints a QR on the LAN address, the TUI syncs at launch and at quit.
6. **Capacitor** wrap for Android (iOS later for free): CORS and a preflight
   on `/v1/*` for the `https://localhost` origin the WebView serves from, a
   server-URL field at login (the page origin no longer identifies the
   server), the Android SDK derivation deleted in `1c9717d`, a signed-APK
   release job.
