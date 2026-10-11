# Changelog

All notable changes to ordo are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `gc` shows this changelog in the TUI

## [0.10.0] - 2026-10-11

Update ordo on every machine that shares the data directory before nesting anything or shipping a version. A project with a sub-category or subtask is saved as schema 2, one with a shipped version as schema 3, and older builds refuse to open both.

phasionary is now ordo. The binary, the Go module, the `ORDO_*` variables and the directories are renamed, and nothing reads the old ones: move `~/.config/phasionary` to `~/.config/ordo` and `~/.local/share/phasionary` to `~/.local/share/ordo` on every machine.

### Added

- Versions: a `vX.Y.Z` category is a version. `s` ships or reopens it; a shipped version folds and dims, and shows a red `! N open` while tasks are left in it. Open versions show done, total and in-progress counts
- `M` sends the task, or a visual range, to an open version or to the `next` inbox
- `A` on a version proposes the next patch version as the new category's name
- `ordo versions [--json]` lists versions with their progress and ship date, and `ordo category ship|unship` ships or reopens one
- One level of sub-categories and subtasks. `>` nests the row under the one above it and `<` un-nests it; `a`, `A` and `-` add at the cursor's depth
- `--parent` on `category add` and `task add`, and `parent/child` paths wherever the CLI takes a category
- Markdown export and import keep the nesting, and indented bullets in a paste become subtasks
- A multi-line paste asks how it lands: one task per line (`t`), one task with the rest as its description (`d`), every line joined into one title (`o`) or appended to the focused task's description (`a`). Indented lines under a pasted task go into its description
- `x` cuts to the system clipboard, and `p` moves the cut task, keeping its ID and any edit made since the cut
- `ze` expands the focused task in place; a focused description is capped at ten rows
- Eight tag marks, a dot and a diamond in each colour. `t` on an untagged task or a visual range picks a mark the category's open tasks don't use, and `T` recolours every task sharing a mark at once
- Renaming a category to a name its siblings already use keeps the editor open and offers to merge the two
- `task_commands` in the config binds a key to a program that acts on the focused task, with an optional one-line prompt
- `--task <id>` opens the TUI on that task; `task edit --description` sets a description, and the CLI shows descriptions
- A Syncthing conflict copy of a project is named when the project opens
- `docs/file-format.md` documents the project file for other apps
- `ctrl+c` quits from every mode, saving a description being edited
- macOS release binaries for Intel and Apple silicon, and a Homebrew tap. On macOS, `gx` opens links with `open`, `p` reads the clipboard with `pbpaste` and `copy_via: wl-copy` copies with `pbcopy`
- The Nix flake builds for aarch64-linux and aarch64-darwin as well as x86_64-linux
- `--version` reports the module version of a `go install` build, and the pseudo-version and commit of a plain `go build`

### Changed

- phasionary is now ordo: the `ordo` command, the `ordo` and `ordo-bin` AUR packages, `ORDO_*` environment variables, `~/.config/ordo` and `~/.local/share/ordo`
- The Go module is `github.com/ilyasturki/ordo`, so `go install github.com/ilyasturki/ordo/cmd/ordo@latest` works
- Copies go to the clipboard through the terminal's OSC 52 by default. `copy_via: wl-copy` uses `wl-copy` instead
- `ctrl+v` in every line input reads the system clipboard
- `p` skips reading the system clipboard while the terminal has kept focus since ordo's last copy
- A hung `wl-copy` or `wl-paste` is killed with its process group after two seconds
- Tags are coloured marks only; labels are gone
- Category names only need to be unique among siblings. Imports, pastes and the CLI keep them unique, and a JSON import suffixes a clashing name
- The expanded help screen lasts for the session instead of being saved in the config
- Only `<id>.json` files whose ID matches the file name load as projects; other files in `projects/` are skipped

### Removed

- The sync server, the web app, the change journal and the NixOS module, along with `phasionary serve` and `phasionary sync`. Syncthing over the plain JSON files replaces them
- The `help_expanded` config key, which is now ignored

### Fixed

- The cursor stays on its item when the tree is rebuilt or the filter changes
- A wrapped title or category name gets the rows its tag and badges take, so it no longer spills past its row
- Renaming a project keeps unsaved edits
- The project opened after a delete keeps the guard against overwriting changes made on disk
- `p` on an empty clipboard says there is nothing to paste instead of calling the clipboard unreadable
- The stale-project message names `r`, the key that reloads
- `category delete` asks before deleting a category that holds only separators
- `task edit` validates every field it sets, as the TUI does

## [0.9.0] - 2026-09-14

### Added

- A self-hosted sync server with `phasionary sync` commands, an embedded web app with keyboard shortcuts and search, and a pairing QR code
- Sync at launch and at quit once a device is enrolled
- A schema version in each project file
- Multi-line pastes become tasks and separators
- `esc` cancels a pending cut
- A full-screen project picker with metadata columns
- The options dialog marks non-default values with a dot, and `d` resets one

### Changed

- The view scrolls by rendered row, so tall titles can be edited and wrap
- The help dialog shows the shortcuts in two layouts on a shared dialog frame

### Removed

- The Android app and its JSON API

### Fixed

- The clipboard finds the Wayland socket when `WAYLAND_DISPLAY` is unset, and falls back to OSC 52

## [0.8.6] - 2026-08-28

### Added

- A description preview under each task
- Separators can be cut and pasted
- The cursor is saved in the background, so a force quit keeps it

### Changed

- Titles have no length cap; the rendered rows are clamped instead

## [0.8.5] - 2026-07-21

### Added

- `space` in visual mode cycles the status of the whole range
- The cursor is restored and centred per project when it reopens
- Description rows can be selected in visual mode
- The Android client edits tasks, categories, folds and separators

### Changed

- `phasionary serve` requires a token by default and keeps `config.json` owner-only

### Fixed

- IDs are validated at the store boundary and imports are checked
- Control characters are rejected and single-line fields are capped
- `state.json` is written atomically and reloaded before each write

## [0.8.4] - 2026-07-13

### Added

- Paste and `ctrl`+arrow word movement in the description editor
- The mouse wheel scrolls the description editor, one line per notch

### Fixed

- Copying from a description row copies the description body
- Wheel scrolling is throttled

## [0.8.3] - 2026-07-10

### Added

- Critical and trivial priority levels

### Changed

- Saves run in the background and projects load faster

## [0.8.2] - 2026-07-09

### Added

- `/` in the help dialog filters shortcuts by key or label
- Visual selections sweep up separators

### Changed

- `enter` saves a description and `shift+enter` adds a newline

### Fixed

- A new task added on a separator lands below it
- `shift+backspace` acts as backspace in text inputs
- Priority icons and tag colours stay visible on the highlighted row

## [0.8.1] - 2026-07-09

### Added

- Separators: divider rows inside a category
- Coloured task tags with labels, filtering and copy-paste
- Type to filter in the project picker

### Fixed

- `zz` centres the cursor consistently
- `alt+enter` works as `shift+enter` to edit a description

## [0.8.0] - 2026-07-02

### Added

- A JSON API under `/api/v1` and a native Android client
- Release builds ship a signed Android APK

### Removed

- The htmx web UI

## [0.7.2] - 2026-06-30

### Added

- `/` searches titles, with `n` and `N` to jump between matches
- `gy` copies a task's ID, URL or title
- The project picker pages, keeps a scroll margin and shows each project's progress
- Visual mode moves a block across category boundaries

### Fixed

- `pick` and `--project` no longer override a directory's linked project

## [0.7.1] - 2026-06-01

### Added

- A login form for the web UI

### Changed

- `phasionary serve` takes separate host and port options

### Fixed

- `import` saves the projects it creates

## [0.7.0] - 2026-05-29

### Added

- `phasionary serve`, a web UI over the same files, guarded by per-project locks, and a NixOS module for it
- An option to expand descriptions by default
- The mouse wheel scrolls the task list

### Changed

- Item info moved to `gi`, the project picker to `ctrl+p` and the time estimate to `ctrl+t`
- Descriptions render as an italic quote block

## [0.6.5] - 2026-05-27

### Added

- `pick`, `link` and `unlink` tie a project to a directory
- `J` and `K` shift a visual selection
- A release workflow that publishes binaries and the AUR packages

## [0.6.4] - 2026-05-26

### Added

- Undo and redo with `u` and `ctrl+r`
- A shortcut bar at the bottom of the screen
- `shift+space` cycles the status backwards
- In visual mode, `o` and `O` swap the anchor

### Changed

- Moved to Bubble Tea, Bubbles and Lip Gloss v2
- Mouse clicks are no longer handled

## [0.6.3] - 2026-05-26

### Added

- Inline task descriptions with an in-app editor
- `r` reloads the project from disk

### Changed

- `status_display` defaults to icons

## [0.6.2] - 2026-05-26

### Added

- Visual select mode
- A priority colour preference in the options dialog

### Changed

- The status moved to the project line

### Removed

- Task sorting

## [0.6.1] - 2026-05-25

### Added

- Task descriptions, exported to markdown and shown as a badge
- A filter hub for status, priority and category
- `J` and `K` move tasks across categories
- `gx` opens a URL from the focused task
- `zt` and `zb` scroll the cursor to the top or bottom
- A scrollable help dialog

## [0.6.0] - 2026-03-18

### Added

- `{` and `}` jump between categories
- A new directory opens the project picker instead of the global last project

### Fixed

- Selection colours adapt to light terminals

## [0.5.1] - 2026-03-05

### Fixed

- A `state.json` inside the projects directory no longer shows up as a project

## [0.5.0] - 2026-03-04

### Added

- `Y` copies a whole category as markdown
- Categories show an aggregate status badge
- Fold state persists across sessions

### Changed

- New tasks are sorted into place after adding
- The status toggle includes cancelled
- The shortcuts line at the bottom is gone

### Fixed

- `?` no longer opens the help while typing
- The cursor and paste in wrapped text fields

## [0.4.0] - 2026-02-03

### Added

- Fold and unfold categories
- Manual project order in the project picker

### Changed

- Completed and cancelled tasks are dimmed

## [0.3.1] - 2026-02-03

### Changed

- Projects are stored in a `projects/` subdirectory

### Fixed

- Custom data paths get the `projects/` subdirectory too

## [0.3.0] - 2026-02-03

### Added

- Time estimates on categories
- Vim-style cut, copy and paste for tasks

## [0.2.1] - 2026-02-03

### Added

- A project per directory, with a global fallback
- `--version` and `-v`
- Shell completions, installed by the Nix package

## [0.2.0] - 2026-02-03

### Added

- New projects start with sample tasks

### Changed

- A flat CLI with short aliases
- Unfocused selection styles for light and dark themes

## [0.1.1] - 2026-02-03

First tagged release: a TUI and CLI over one JSON file per project, with categories, priorities, statuses, time estimates, a project picker, markdown import and export, and a Nix flake.
