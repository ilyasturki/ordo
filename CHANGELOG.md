# Changelog

All notable changes to ordo are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `l` in the project picker, dialog or full screen, unlinks this directory on the row marked `(linked)`, and the footer reads `l unlink` there. The unlink holds for the rest of the session: opening or creating a project no longer links the directory until `l` links it again

### Changed

- `enter` or `space` on a category's `(N hidden by the filter)` row shows those tasks in place, marked `(filtered out)`, until the cursor leaves the category, where it used to clear the whole filter. The filter and its saved copy stay as they were; `f` → Clear all filters still clears it
- The project picker dialog (`ctrl+p`) has the rounded border of the other dialogs, so it stands out from the list behind it. The full-screen picker stays frameless

### Fixed

- Creating a project from the picker, or deleting the current project, no longer replaces the directory's link to another project, and deleting the linked project drops its link

## [0.13.0] - 2026-10-29

### Added

- An Android app, attached to each release as an APK. It keeps a copy of every project and a queue of the edits made on it, so it opens at once and works without signal, and the queue reaches the project files through `ordo serve` once the connection is back. The layout follows Google Tasks. Projects are tabs, each category is a card with its own `+` and Completed row, and the task page lists its fields in rows that open their choices
- The app pairs once, by scanning the QR code `ordo serve pair` prints or pasting its link. A long press lifts a row to drag it, and sliding it right nests it under the task above. A sideways swipe on the list changes project, a tap plays the system click sound, a long press buzzes, and Back closes the open sheet or page before it leaves the app
- The app manages categories and projects. A category's ⋮ menu renames it, adds a sub-category or deletes it, New category ends the list, New project follows the tabs, and the top bar's ⋮ renames or deletes the open project. A project made in the app starts empty
- Search and Filter in the app's top bar. The filter picks statuses, priorities, tags and categories, and the app keeps it per project
- Every edit in the app offers Undo in its snackbar, a delete included, offline too
- An open app syncs as soon as a project file changes on the server, from the TUI, Syncthing or another device
- [`docs/sync.md`](docs/sync.md) documents the `/api` the app syncs through, for any other client

### Changed

- `ordo serve` serves the new app to a browser in place of the htmx pages, and a browser paired before stays signed in. A build without the app bundle shows a browser a placeholder page instead and still syncs the Android app. That covers `go install`, the AUR source package and the nix package outside x86_64-linux; the release binaries, Homebrew, `ordo-bin` and the x86_64-linux nix package carry the app
- Undo takes an edit back on the app's own copy and queues the change, so it works offline and after another device's edit, which the htmx app's undo refused

### Removed

- The htmx web app, with its drawer, its Task button and its PWA manifest

## [0.12.1] - 2026-10-22

### Added

- `l` in the project picker links this directory to the highlighted project and opens it, so the next launch here opens it too. The picker marks the linked project `(linked)`
- While a filter hides some of a category's tasks, the category ends with an `(N hidden by the filter)` row the cursor stops on, and `enter` or `space` there clears the filter. It replaces the `(no tasks)` such a category showed, and the shortcut bar reads `f filtered` while a filter is on
- The TUI remembers each project's filter across restarts, in this machine's `state.json`. A new project starts unfiltered, and the web app keeps its own filter
- The web app's sign-in page pairs with a code: Scan code reads the QR code with the camera inside the app, and Pairing link or code takes a pasted link or code. Without a camera it says why and offers the paste field
- Devices → Pair a new device shows a one-time pairing QR code and link on a device already signed in, after the passphrase when one is set. Devices shows which device paired which
- The sign-in page moves on by itself once the device is paired in another tab, and the pair page opened in a browser tab suggests pasting the link into the installed app instead
- Touch gestures in the web app: a swipe from the left edge opens the drawer and a sideways swipe on it or outside it closes it, both following the finger and finishing with its speed; a swipe down on a bottom sheet's handle, or anywhere on it while its content is at the top, closes it; a sideways swipe dismisses the snackbar

### Changed

- A task an edit takes out of the filter stays where it is, marked `(filtered out)`, until the cursor leaves it, so `space` can walk it through every status under a status filter. Status, priority and tag edits all behave this way, in visual mode too, where the rows go once visual mode ends and the cursor moves off them. Before, the tag picker and a pasted tag hid the task at once, and the other edits left it unmarked until the next fold or move
- Revoking a device, from Devices or `ordo serve revoke`, also signs out the devices it paired, and lists them; `ordo serve devices` shows which device paired each
- The web app folds finished tasks into a "Completed (n)" row at the bottom of each category, closed until you open it, and remembers it open. A task goes there once it and all its subtasks are done or cancelled; a finished subtask of an open task stays in place, struck through. The Open, Done and All chips are gone, and separators show again
- A web task's circle still steps through To do, Doing, Done and Cancelled, but the row changes at once and stays put a second after the last tap, then slides into Completed once the task is finished, or back out of it. Undo stays in the snackbar
- The web app no longer shows progress the terminal app lacks: no bar or "N of M done" under a category, no percentage or done count on the projects page. A category keeps its `~` estimate
- A web row shows the first two lines of its task's details in grey, without the images, instead of a notes icon
- `+` on a web category opens an add row at the top of that category with the keyboard up, in place of the New task sheet. Enter adds the task there and opens the next row below it, so a batch keeps its typing order and the keyboard stays up between tasks. Escape or back closes the row; tapping away adds what it holds, then closes it. The Task button opens the row in the category at the top of the screen
- A web category's ⋮ menu offers Add separator, which puts one at the top of the category, Fold, and Select completed, which selects its finished tasks for the bulk actions. With one task selected, the edit sheet offers Separator below, like `-` in the terminal. A new separator opens its label in place, and leaving it empty keeps a bare rule; tapping a separator edits its label
- An empty category on the web reads "No tasks yet", and one whose tasks are all finished "All tasks complete" above its Completed row
- The web app's task page is laid out like Google Tasks: the category name in the top bar moves the task, and `⋮` holds Move up, Move down, Nest or Un-nest and Delete. Status, priority, tag and estimate are one row each showing the current value, and tapping a row opens a sheet with the choices. Details and images follow, then the subtasks with an "Add subtask" row, and a "Mark done" or "Mark open" button sits at the bottom. The created and updated dates are gone from the page
- Dragging in the web app works like Google Tasks. A long-press lifts the row, with a haptic tap where the browser allows, and moving it drags it; sliding it about 40px right nests it under the task above and left un-nests it, the gap it leaves showing the indent before the drop. Letting go without moving selects the row, in place of the old long-press timer that cut drags short. A row no longer nests by accident, a place the drop can't keep isn't offered, an empty category takes a drop, the list scrolls at the screen's edge, a drop never opens the task, and a live refresh waits for the finger to lift. A mouse still drags at once, and Escape cancels

### Fixed

- `y` on a separator copies it as `### label`, or `---` for a bare one, so `p` pastes a separator, here or in another ordo, instead of a task, or instead of the task copied before. `gy` on a separator offers the plain label
- `x` and `d` on a lone separator take it the way they take a task, so `p` puts it back, label and all. `x` cuts it to the clipboard as `y` would copy it, where it used to refuse with "Can only cut tasks", and `d` keeps it for `p` where it used to drop it
- A visual `y` or `Y` copies the separators in the range as those same lines, in order, so pasting the text into another ordo keeps them
- Visual mode from a task moves onto folded categories instead of stopping above them, and never unfolds them. A folded header in the range stands for its whole category, sub-categories included: `y`, `Y`, `x`, `d` and `m` take its tasks and separators along, status and tag edits leave them alone, and the status line counts them (`1 task + 1 folded category (2 tasks) selected`). A range that ran past folded categories used to leave their tasks out. `J`, `K`, `>` and `<` refuse such a range
- The visual delete prompt no longer counts separators as tasks
- A filter on a category that was then deleted, from the TUI, the CLI or another machine, hid every task until "Clear all". The filter now drops a category that no longer exists
- The TUI no longer drops edits silently after the phone, the CLI or Syncthing changed the project first. A refused save marks the project `[changed on disk: r drops N edits]` until `r` reloads it, the mark staying when the status line clears; `q`, `ctrl+c`, switching projects and creating one ask before dropping those edits, and quitting anyway exits with status 1, naming them on stderr
- A save still queued when `r` reloads can no longer land after the reload and overwrite the change it showed: `r` waits for it, and each save is checked against the load it came from
- After a cut whose paste was refused, `r` gives the cut back, so `p` moves the rows instead of leaving a second copy
- `ctrl+c` in the inline title, category or add editor saves what was typed before quitting, as the description editor already did, instead of dropping it. An edit the editor refuses, like a name another category has, keeps the editor open on the first `ctrl+c`, and a second one quits without it. When the project already holds edits that couldn't be saved, it asks first, and `n` goes back to the editor
- `ordo task add` and `ordo task edit` take `-e 2d`, a day being 8 hours, so the CLI reads back the `2d` it and the web app print. It refused any day estimate before
- On the phone, a title or details edit refused because the task changed elsewhere no longer loses the typed text to Reload: the field keeps it and offers Keep mine or Use theirs, and a refresh, an action or leaving the page and coming back keeps it too
- On the phone, text typed into a task another device deletes is no longer lost: "That task is gone" offers Copy, which puts the text on the clipboard. The offer comes back after any other message and after a reload, until Copy takes the text. The open page moves to its project instead of reading "That didn't work"
- Saving a task's details of a few KB on the phone no longer fails behind a reverse proxy: the web app's answers carried the whole text in a response header, which nginx refuses past its 4 or 8 KB buffer. They now carry only flags
- The web app's drawer and bottom sheets slide away when they close, by a tap outside, Escape, Android back or a drawer item, instead of vanishing in one frame, Firefox for Android included. A drawer item closes the drawer before the next page shows, and the current project just closes it. The task page's `⋮` menu fades out the same way
- Touch in the web app: the page no longer scrolls behind an open drawer or sheet, a sideways swipe no longer goes back a page, chips, buttons and segments answer a 48px touch, and a long-press on a row, a separator or a drawer item no longer selects text or opens the link menu
- The web app's Task button steps aside while a category's `+` or `⋮` sits under it, so a tap on a header at the bottom of the screen reaches that category instead of adding to the one at the top
- A web page swap no longer trips the Content-Security-Policy: htmx stopped copying the old page's inline styles, such as the task title's height, onto the new one. `testdata/web/run.sh` now fails on any CSP violation

## [0.12.0] - 2026-10-19

### Added

- `ordo serve`, a Material web app for the phone over the same project files: step a task's status from the list, edit its title, details, priority, tag, estimate and category, attach photos, add tasks, subtasks and separators, long-press to select several and edit them together, drag to reorder or nest, fold categories, filter open or done tasks, and undo from the snackbar. An open page refreshes when the TUI, the CLI or Syncthing changes the file, and a title or details edit made on a stale page is refused instead of overwriting the newer one
- `ordo serve pair` prints a one-time link and QR code that pairs a device, which then stays signed in while it's used at least once a year; `ordo serve passphrase` sets a passphrase to sign in without a link, and `ordo serve devices` and `ordo serve revoke` list and sign out devices
- A NixOS module, `nixosModules.default`, runs `ordo serve` as a hardened service

## [0.11.0] - 2026-10-11

### Added

- `gc` shows this changelog in the TUI
- A "Category Counts" option (`show_category_counts`, off by default) shows done / total on every top-level category that holds tasks, a parent counting its sub-categories
- `ctrl+a` and `ctrl+x` add or take one from the patch of the version in a category's name, on the focused category or a visual range of them, and `gm` and `gM` bump its minor or major. Only the digits change, and a name a sibling already has is refused
- Images on tasks: `p` with an image on the clipboard attaches it to the focused task, and so does dropping image files on the terminal. In the description editor, `ctrl+v` and a drop insert it at the cursor. The image is copied into `projects/attachments/`, named by its content, and the description references it as `![name](attachments/<hash>.png)`
- An image reads `▣ name` in the description, or `▣ name (not synced yet)` before its file arrives, and the title row counts them as `▣` or `▣2`, with descriptions expanded too. `gi` lists each image's path, size and pixel size, and `gy` offers its path
- `gx` opens the URLs in the description as well as the title's, and opens an image in the desktop's viewer
- `ordo attachments prune` lists the images no project references any more and that are over 30 days old, conflict copies counted; `--yes` removes them. Nothing else ever deletes an image
- Text that leaves ordo points an image at its absolute path, so an agent handed it can open the file: `y` on a task, `Y` on a category and `ordo export -f markdown`. A visual `y` or `Y` lists each task's images under its bullet. The project file keeps the relative reference, and `ordo export -f json` too
- `ordo task attach <task> <file>` attaches an image from the command line, and `-` reads it from stdin: `wl-paste -t image/png | ordo task attach <task> -`. `--name` names it
- `ordo task show` lists the task's images with their absolute path, marking a missing one, and `ordo attachments path` prints the directory they live in
- Task commands get `ORDO_TASK_IMAGES`, the absolute paths of the task's images, one per line

### Changed

- Folded categories in a row stack one header per row, with no blank line between them and no `(folded)` row. Folded sub-categories under the same parent stack at their indent
- Category headers no longer show counts unless "Category Counts" is on, versions included, and the `· N in progress` suffix is gone: the `[/]` badge says it
- `M` is now `m` and moves the task, or a visual range, to any category instead of an open version, with `/` to filter the list

### Removed

- Shipping a version: `s`, `ordo category ship|unship`, the ship date on its header, the dimming and auto-folding of shipped versions and the red `! N open`. A file a 0.10.0 build saved with a ship date still opens, and the date is dropped on the next save
- `ordo versions`. `ordo categories` lists the same task counts
- The `next` inbox. A category named `next` is an ordinary category again

### Fixed

- `A` on a version adds an empty category right after it again, like on any other category, instead of proposing the next patch at the top or the end of the project
- Moving onto the row just above the last one of a scrolled project shows it again. A category added or pasted at the end, a folded last category and the second-to-last task were left under `↓ more below`
- Status messages and the visual-mode count stay on screen once the project line scrolls off: they sit on the `↑ more above` row instead
- The view follows the focused row when a cut mark, a priority or a tag makes it wrap onto one more row at the bottom of the screen
- `ctrl+d` and `ctrl+u` move half a screen of rows, so wrapped titles and open descriptions no longer make them skip past a whole screen
- `zb` no longer leaves the row one line below the screen when the view lands one row from the top
- `p` and `ctrl+v` with an image on the clipboard no longer paste its bytes as hundreds of garbage tasks or characters

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

[Unreleased]: https://github.com/ilyasturki/ordo/compare/v0.13.0...HEAD
[0.13.0]: https://github.com/ilyasturki/ordo/compare/v0.12.1...v0.13.0
[0.12.1]: https://github.com/ilyasturki/ordo/compare/v0.12.0...v0.12.1
[0.12.0]: https://github.com/ilyasturki/ordo/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/ilyasturki/ordo/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/ilyasturki/ordo/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/ilyasturki/ordo/compare/v0.8.6...v0.9.0
[0.8.6]: https://github.com/ilyasturki/ordo/compare/v0.8.5...v0.8.6
[0.8.5]: https://github.com/ilyasturki/ordo/compare/v0.8.4...v0.8.5
[0.8.4]: https://github.com/ilyasturki/ordo/compare/v0.8.3...v0.8.4
[0.8.3]: https://github.com/ilyasturki/ordo/compare/v0.8.2...v0.8.3
[0.8.2]: https://github.com/ilyasturki/ordo/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/ilyasturki/ordo/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/ilyasturki/ordo/compare/v0.7.2...v0.8.0
[0.7.2]: https://github.com/ilyasturki/ordo/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/ilyasturki/ordo/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/ilyasturki/ordo/compare/v0.6.5...v0.7.0
[0.6.5]: https://github.com/ilyasturki/ordo/compare/v0.6.4...v0.6.5
[0.6.4]: https://github.com/ilyasturki/ordo/compare/v0.6.3...v0.6.4
[0.6.3]: https://github.com/ilyasturki/ordo/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/ilyasturki/ordo/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/ilyasturki/ordo/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/ilyasturki/ordo/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/ilyasturki/ordo/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/ilyasturki/ordo/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ilyasturki/ordo/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/ilyasturki/ordo/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/ilyasturki/ordo/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/ilyasturki/ordo/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/ilyasturki/ordo/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/ilyasturki/ordo/releases/tag/v0.1.1
