# Phasionary

A terminal-first project planner for people who'd rather press `j` than reach for a mouse. Single binary, plain JSON on disk, no accounts, no network.

![Phasionary demo](assets/demo.gif)

## Install

### Arch Linux (AUR)

```bash
yay -S phasionary       # build from source
yay -S phasionary-bin   # prebuilt binary
```

### Nix

Run without installing:

```bash
nix run github:ilyasturki/phasionary
```

Or pin it in your system flake:

```nix
inputs.phasionary.url = "github:ilyasturki/phasionary";
# environment.systemPackages = [ inputs.phasionary.packages.x86_64-linux.default ];
```

### Prebuilt binary

Grab the latest `phasionary-linux-x64` or `phasionary-linux-arm64` from the [releases page](https://github.com/ilyasturki/phasionary/releases), `chmod +x`, drop in `$PATH`. The sync server ships alongside as `phasionary-server-linux-*`.

### From source

```bash
go build -o phasionary ./cmd/phasionary
go build -o phasionary-server ./cmd/phasionary-server   # optional, see Sync
```

## Quick start

```bash
phasionary                # Launch TUI — opens the last project, or the picker
phasionary -p "Life"      # Jump straight into a project by name
phasionary -p "Life" --task <id>   # Open with the cursor on a task
```

First run seeds a default project. Inside the TUI: `a` adds a task, `Space` cycles status, `i` opens task info, `?` shows every keybinding.

## CLI

Every TUI action has a CLI counterpart, useful for scripts and quick edits. All commands accept `-j` for JSON output.

```bash
phasionary tasks -s todo -C Feature                   # Filter tasks
phasionary task add -C Feature "Build widget"         # Capture
phasionary task status <id> in_progress               # Triage
phasionary export -f json -o project.json             # Snapshot
phasionary import project.md                          # Restore from markdown
```

Run `phasionary --help` for the full surface (projects, categories, config, completions).

## Sync (optional)

Phasionary is local-only and stays that way until you run a `phasionary-server` of your own. The server is what puts the [web app](#web-app) and your other machines in front of the same projects; if the TUI on one machine is all you want, you never need it.

On the machine hosting the server:

```bash
phasionary-server            # listens on 0.0.0.0:7777, enrolls this machine, prints a QR
```

It enrolls the local phasionary itself and uploads its projects — the machine holding the database has nothing to prove, so there is no code to copy — then prints a QR for the next device. That enrollment outlives the server process: from then on the TUI on that machine syncs with it, and `phasionary sync logout` is the way back to local-only, which is what you want after trying the server once on a laptop. Every start says so while the machine is enrolled; `--no-local-enroll` skips it altogether. Scan it from a phone and the web app opens already enrolled. `phasionary-server pair` prints another one whenever you need it; the QR only ever goes to a terminal, never to a log. On another computer, use the address it prints:

```bash
phasionary sync login http://server:7777   # asks for the code, uploads local projects
phasionary sync now                        # push local changes, pull everyone else's
phasionary sync status
phasionary sync logout                     # back to local-only
```

Once enrolled, the TUI pulls when it starts and pushes when you quit, so the phone is never waiting on a forgotten `sync now`. Both are capped at three seconds: an unreachable server delays a session, it never blocks one. When the server cannot be reached the project line carries an `[offline]` tag for the session and the changes wait in the outbox; a server that answers with an error shows the error instead. Nothing else touches the network.

Stop any file syncer (Syncthing, Dropbox, …) carrying `~/.local/share/phasionary` before enrolling: two pipes moving the same files fight each other. The device identity and token live in `~/.local/state/phasionary/`, which must never be copied between machines. If a sync changes a project the TUI has open, the TUI's next save is refused until you press `R` to reload.

The server listens on every interface, because a phone cannot reach loopback. It speaks plain HTTP and every request carries the device's bearer token, so keep it on a network you trust (your LAN, Tailscale, WireGuard) or behind a TLS reverse proxy — never straight on the internet. `--host 127.0.0.1` closes it back down to the local machine.

Reaching it by a hostname rather than an IP needs that name allowed — `phasionary-server --allowed-host phas.example.net`, or `services.phasionary-server.allowedHosts` on NixOS — otherwise every request is refused with 421. IP addresses and `localhost` always pass; a name has to be listed, which is what stops another site from pointing its own hostname at your server and reading the answers.

### NixOS

```nix
imports = [ inputs.phasionary.nixosModules.phasionary-server ];
services.phasionary-server = {
  enable = true;
  host = "100.64.0.1";   # e.g. a Tailscale address; default 127.0.0.1
  openFirewall = true;
};
```

Then mint codes with `sudo -u phasionary phasionary-server --data /var/lib/phasionary-server pair`. The module keeps the default bind at `127.0.0.1` and runs the service with `--no-local-enroll`: a box serving other people's devices has no phasionary of its own to enroll.

## Web app

The server also serves a web app at its own address — the same projects, the same glyphs and colors as the TUI, on a phone. Scan the QR from `phasionary-server pair` and it opens enrolled, downloads your projects and keeps working offline; changes queue locally and go up when the server is reachable again. Typing the code into the app by hand does the same thing.

It can create, edit, delete and reorder projects, categories and tasks, including separators, and search both lists. On a keyboard the basics carry over — `j`/`k`, `space` to cycle a status, `⏎` to edit, `a`/`A` to add, `/` to search; `?` lists them. Filtering, visual mode, undo and copy/paste stay in the TUI. When a sync fails, a banner stays up until one succeeds; otherwise sync is silent.

Installing the app to a home screen, and having it open with no network, needs HTTPS — browsers only register a service worker on a secure origin, so put the server behind a TLS reverse proxy if you want that. Over plain HTTP it still works, it just has to be loaded from the server each time.

Building it needs Node; `nix build .#phasionary-server` does it for you, and so does `just build`. A plain `go build` produces a server that answers `/v1` normally and reports the missing bundle on every other path.

The architecture is laid out in [`docs/sync-design.md`](docs/sync-design.md).

## Configuration

Config lives at `~/.config/phasionary/config.json`. Override with `phasionary config set <key> <value>` or edit directly.

| Key | Values | Default | Description |
|-----|--------|---------|-------------|
| `status_display` | `text`, `icons` | `icons` | How task status renders in the TUI |
| `default_project` | project UUID | (none) | Project to open on launch |
| `priority_color` | `full`, `icon`, `none` | `full` | How priority is colored — full row, icon only, or off |

Paths can be relocated with `PHASIONARY_CONFIG_PATH` and `PHASIONARY_DATA_PATH` — both name a directory; `PHASIONARY_CONFIG_PATH` also accepts a path ending in `config.json` for convenience.

## Data

One project per JSON file under `~/.local/share/phasionary/projects/{uuid}.json`. UI state (fold state, cursor position, last project per directory) sits next to it in `state.json`. Every change is written synchronously — no undo, but nothing is ever in-flight.

## License

[MIT](LICENSE)
