# Phasionary

A terminal-first project planner for people who'd rather press `j` than reach for a mouse. Single binary, plain JSON on disk, no accounts, no network.

![Phasionary demo](assets/demo.gif)

[![CI](https://github.com/ilyasturki/phasionary/actions/workflows/ci.yml/badge.svg)](https://github.com/ilyasturki/phasionary/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ilyasturki/phasionary)](https://github.com/ilyasturki/phasionary/releases/latest)
[![AUR version](https://img.shields.io/aur/version/phasionary)](https://aur.archlinux.org/packages/phasionary)
[![License: MIT](https://img.shields.io/github/license/ilyasturki/phasionary)](LICENSE)

## Install

Phasionary runs on Linux and macOS.

### Homebrew

```bash
brew install ilyasturki/phasionary/phasionary
```

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

Or pin it in your system flake (x86_64-linux, aarch64-linux and aarch64-darwin):

```nix
inputs.phasionary.url = "github:ilyasturki/phasionary";
# environment.systemPackages = [ inputs.phasionary.packages.${pkgs.stdenv.hostPlatform.system}.default ];
```

### Go

```bash
go install github.com/ilyasturki/phasionary/cmd/phasionary@latest
```

### Prebuilt binary

Grab `phasionary-linux-x64`, `phasionary-linux-arm64`, `phasionary-darwin-x64` or `phasionary-darwin-arm64` from the [releases page](https://github.com/ilyasturki/phasionary/releases), `chmod +x` it and put it in your `$PATH`. On macOS, a binary downloaded with a browser also needs `xattr -d com.apple.quarantine <file>`.

### From source

```bash
go build -o phasionary ./cmd/phasionary
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
phasionary category add ui --parent v0.2.0            # Nest a sub-category
phasionary task add --parent <id> "Pick a default"    # Nest a subtask
phasionary tasks -C v0.2.0/ui                         # Address it by path
phasionary task edit <id> --description "Details"     # Describe
phasionary task status <id> in_progress               # Triage
phasionary versions                                   # Progress and ship date per vX.Y.Z category
phasionary category ship v0.2.0                       # Mark a version shipped (unship reopens it)
phasionary export -f json -o project.json             # Snapshot
phasionary import project.md                          # Restore from markdown
```

Run `phasionary --help` for the full surface (projects, categories, config, completions).

## Configuration

Config lives at `~/.config/phasionary/config.json`. Override with `phasionary config set <key> <value>` or edit directly.

| Key | Values | Default | Description |
|-----|--------|---------|-------------|
| `status_display` | `text`, `icons` | `icons` | How task status renders in the TUI |
| `default_project` | project UUID | (none) | Project to open on launch |
| `priority_color` | `full`, `icon`, `none` | `full` | How priority is colored — full row, icon only, or off |
| `copy_via` | `osc52`, `wl-copy` | `osc52` | How copies reach the system clipboard — the terminal's OSC 52, or `wl-copy` |

### Task commands

`task_commands` binds keys in the TUI to programs that act on the focused task, such as filing it in another tool. None are set by default.

```json
{
  "task_commands": [
    { "key": "gn", "label": "notify", "prompt": "message", "run": ["notify-send", "--app-name=phasionary"] }
  ]
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `key` | yes | One character (`m`), or `g` or `z` followed by one (`gn`). A key a built-in shortcut already uses is refused at launch. |
| `label` | yes | Name shown in `?` and the shortcut bar. |
| `prompt` | no | Asks for one line first; Enter on an empty line cancels. |
| `run` | yes | Program and arguments, run directly, never through a shell. |

The command runs in the background. It gets the task through environment variables: `PHASIONARY_PROJECT_ID`, `PHASIONARY_PROJECT_NAME`, `PHASIONARY_TASK_ID`, `PHASIONARY_TASK_TITLE`, `PHASIONARY_TASK_DESCRIPTION`, `PHASIONARY_CATEGORY` (its path, `parent / name` for a sub-category, which `-C` resolves), `PHASIONARY_CATEGORY_ID` and `PHASIONARY_INPUT` (the prompt's answer). The first line it prints shows in the status line. A non-zero exit shows as a failure with the first line of its error output.

Paths can be relocated with `PHASIONARY_CONFIG_PATH` and `PHASIONARY_DATA_PATH` — both name a directory; `PHASIONARY_CONFIG_PATH` also accepts a path ending in `config.json` for convenience.

## Data

One project per JSON file under `~/.local/share/phasionary/projects/{uuid}.json`. UI state (fold state, cursor position, last project per directory) sits next to it in `state.json`. Edits are saved in the background as you make them, and `u` / `ctrl+r` undo and redo them within a session. [`docs/file-format.md`](docs/file-format.md) documents the format for any other app that reads or writes these files.

## License

[MIT](LICENSE)
