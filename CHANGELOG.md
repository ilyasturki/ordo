# Changelog

All notable changes to ordo are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- An outside change now merges into the open project instead of refusing your edits, whichever writer made it: an edit committed under an open input, or one a sync tool replaced after it saved, stays, row by row and field by field, with no prompt. Only one field changed to two values asks: `m` keeps yours, `t` theirs, `b` both, with yours as a copy just below, `M` and `T` the rest, and `u` takes it back. An edited row outlives a delete. `r` keeps your edits, undo survives a change from outside, and what a choice could discard stays in `recovery/`. See [`docs/file-format.md`](docs/file-format.md#edits-from-outside)

## [0.14.2] - 2026-11-04

### Added

- Pinned categories: `gp` pins or unpins the category under the cursor, and the app's category menu has Pin and Unpin. A pinned category comes first among its siblings, in the list, the pickers, the filter and `ordo categories`, marked with a faint `↑` in the TUI and a pin in the app. A pinned category takes its sub-categories along, and a pinned sub-category goes first under its parent. Moves, adds and pastes keep pinned ones first: `J` and `K` stop at the end of the pinned run. Older builds keep the order but drop the pin. See [`docs/file-format.md`](docs/file-format.md#pins)
- `gA` adds a sub-category: on a category or any row in it, a new last sub-category of it; on a sub-category, a new last sibling. The parent unfolds and the name editor opens, as with `A`
- Notes: `_` adds a line of free text below the focused row, with no status or priority, and opens it for typing. It wraps like a title, moves, cuts, copies and deletes like a separator, and hides under a filter. It copies as `> text`, and a pasted or imported `> ` line comes back as a note. The app shows notes in the list, adds one from a category's menu, and drags, edits and deletes it like a separator. A project holding a note is saved as schema 4, which older builds refuse to open: update ordo on every machine, `ordo serve` and the phone before adding the first one. See [`docs/file-format.md`](docs/file-format.md)
- The readline keys in every text field: `ctrl+_` (or `ctrl+/`) undoes the last change, a typed word at a time, with the cursor back where it was; `ctrl+y` yanks what `ctrl+w`, `alt+backspace`, `alt+d`, `ctrl+k` or `ctrl+u` killed, in the same field or another; `ctrl+t` and `alt+t` swap characters and words; `alt+u`, `alt+l` and `alt+c` change a word's case; `alt+<` and `alt+>` go to the start and end. The help filter keeps `ctrl+d` and `ctrl+u` for paging
- `F` turns the filter off and back on and keeps what it holds: off, every task shows, with no `[filtered]` and no hidden rows, and a change in `f` turns it on again. The app's filter sheet has the same switch at its top, and its filter button shows when the filter is off. Both keep the switch in the project's view file; an older TUI, app or `ordo serve` shows such a filter on and turns it back on when it changes it. See [`docs/file-format.md`](docs/file-format.md#views)
- `ordo changelog` prints the changelog, to pipe into a pager, next to `gc` in the TUI

### Changed

- Naming a category like one of its siblings opens a chooser instead of the merge offer: `t` names this one with the next free number, as in "Fix (2)", `o` gives that number to the other one, `m` merges as before, and `esc` goes back to the name. In the app, the add and rename sheets offer the first two, or Cancel
- `J` and `K` move a sub-category past its parent's edge: `K` on the first one makes it the last sub-category of the category above, `J` on the last one the first of the category below, unfolding it. The first and last top-level categories still stop it, and so does a sub-category of the same name there
- The app's list shows a task title on two lines at most, ending in an ellipsis, and a description preview on one line under a title that wraps. The task page still shows the whole title
- The app's snackbar sits on the bottom edge and the `+` (or "Mark done") rises above it while it shows. A status tap in the list or on a subtask no longer snacks unless it lands on done, which still offers Undo, and every snack lasts 4 seconds
- A tap on a category in the app's categories page goes to it: the page closes, the category unfolds, the list scrolls to it and it stays the top bar's category and the `+`'s target until you scroll. Rename, delete, sub-category and separator move behind the row's ⋮. A category the filter hides says so instead
- No priority and medium are one level, Medium, between High and Low in the TUI filter, the info view, the app's pickers and filter, and the CLI, where `--priority medium` and `none` both match it. A medium task carries no `priority` in its file, so the first save rewrites every medium task of an older file, and the markdown export drops the `(medium)` suffix
- The app's add sheet fits on one line under the title: the category, then icons for the description, priority, colour and estimate, then Save. A set value shows in place of its icon, as the priority's arrows, the colour's dot or the estimate, and a long category name shortens instead of pushing Save onto a second line
- The TUI's filter dialog shows each status and priority with the mark its rows draw, `/`, `x`, `▲`, `▼`…, in a column, following the status display and priority colour options. The app's filter sheet shows the status and priority icons on its chips, and a selected chip keeps its icon or tag dot, its fill saying it is on
- The app's Settings: the Version row opens the changelog, offline, and the Server and Sync rows, which did nothing, are gone; the server shows under Devices as "Signed in to …"
- `just run` runs from source on a fresh copy of your data and config, so a dev build never writes the real files; `just run-real` keeps the old behaviour
- A new app icon: three white bars climbing to the right on indigo, larger on the home screen, whole as a themed icon, and on orange for the Ordo Dev build. The README carries it too
- The app looks more finished: a task's values are the same outlined chips in the list, the add sheet and the task page, legible in dark mode; the task page's image button sits in its top bar, and its subtasks line up with the field that adds them; sheet actions are full-width rows; every empty screen shows its line in the middle; the top bar tints as the list scrolls under it instead of a fixed line; and a ⋮ on Home holds the project's rename, categories, new category, delete and Settings, the tab gestures staying as shortcuts

### Fixed

- The `gc` changelog dialog no longer ends with the list of comparison links
- `alt+d` on a word's last letter in a title, a name, the search or a filter deletes that letter alone, no longer the next word with it
- A row dragged in the app lands where you let it go instead of snapping back: past the line counting what the filter hides, while the previous drop is still saving, back to where it started, or into a category the filter hides, where it stays in sight. A row dropped under a folded category lands first in it and unfolds it, and the confirming buzz comes only once the move is saved. The list scrolls when a row is held just above the navigation bar, follows the row under your finger even when a task is held by one of its subtasks, and no longer scrolls back when you lift the top row of a scrolled list. On the categories page, a category dragged back where it was stays there

## [0.14.1] - 2026-11-01

### Added

- A long press on a project tab or a category header in the app opens that project's categories on a page of their own, as a tree. Drag a row to reorder it among its siblings, its sub-categories with it, and every move offers Undo; a tap opens the menu the header's tap does, and the `+` adds a category. On another project's tab the app goes to that project first. Tapping the current tab still opens the project's menu. The move reaches `ordo serve` as a new `category.place` op in [`docs/sync.md`](docs/sync.md)
- Folds in the app: a chevron folds a category or a task with subtasks, as `za` does in the TUI. A folded category shows its status, how many of its tasks are finished and its estimate, and hides its sub-categories; a folded task keeps its subtask count. A tap on the header still opens its menu, search looks through folds, and adding to a folded category unfolds it

### Changed

- The TUI notices an outside change to the open project's file within a second, without a keypress. With no edit waiting it reloads by itself, keeps the cursor on the same row and says `Changed on disk: reloaded`; a dialog or an input left open waits until it closes. With an edit waiting or refused, `[changed on disk: r drops N edits]` shows at once instead of at the next save, and `r` and the quit confirmations work as before
- The `(N hidden by the filter)` row stays while it shows those tasks and reads `(hide the N filtered out)`: `enter` or `space` on it hides them again with the cursor on it, and so does `esc` from anywhere in the category. Leaving the category still hides them, and the saved filter never changes
- The project picker lists the most recently used project first, where it was alphabetical or in a hand-made order. A project is as recent as the later of its last open on this device and its last edit anywhere, one never opened here counts by its last edit, and a tie falls to the name. `J` and `K` no longer reorder the picker, and `state.json` drops `project_order` for `last_opened`, so an older file still loads. `ordo project list` and completion stay alphabetical
- The app's project tabs follow the same order, by the later of the last time you had a project open or edited it on the phone and its last edit anywhere. The order is set when the app starts and each time it comes back to the front, and stays put in between: opening, editing or syncing a project never moves a tab under your finger, and a project that appears meanwhile goes last
- The Android app draws a separator like the terminal does, a thick rule with a short lead before a bold label, in full contrast
- The Android app's add sheet and task page say "Priority" and "Colour" beside their icons until a value is set, the colour control takes a palette icon instead of a label shape, and the priority menu shows each level's arrows
- The app's top bar has two lines: the name of the category taking most of the screen with the search, filter and settings buttons, then the project tabs on their own line, at a larger size. Settings is a gear instead of an account icon
- The app's `+` opens the add sheet on the category the top bar names, not on the last one used, and the sheet can still change it. A category the filter hides falls back to the first one it shows
- Folds, the filter and which directory opens which project now travel with `projects/`: `<id>.view.json` for each project and `.directories.json` for the links, with directories under your home written from `~`. The first run moves them out of `state.json` and leaves what another device already wrote alone. A change arriving applies in an open TUI within a second, with the cursor on its row. Conflict copies of these files are ignored, as Syncthing reports them. See [`docs/file-format.md`](docs/file-format.md)
- The app shows the same filter as the TUI: a change on either side reaches the other through `ordo serve` with no project edit, last writer wins, and one made offline goes out when back. The phone's own filters move over once, unless a file on the server already has one. The current project stays per device. Older apps and servers keep working without it. See [`docs/sync.md`](docs/sync.md#views)

### Removed

- The status-line notice about Syncthing conflict copies. Ordo ignores them; reporting them is Syncthing's job

### Fixed

- The app's name and add sheets rise with the keyboard in one motion: the keyboard no longer covers them before they jump above it

## [0.14.0] - 2026-10-31

### Added

- `l` in the project picker, dialog or full screen, unlinks this directory on the row marked `(linked)`, and the footer reads `l unlink` there. The unlink holds for the rest of the session: opening or creating a project no longer links the directory until `l` links it again
- A Row marks section at the end of `?` names what each glyph on a row means: status boxes, priority arrows, the tag dot, `¶`, `▣`, the estimate, subtask progress, `✂`, separators, folds and `(filtered out)`. The help filter `/` finds them by meaning

### Changed

- **Breaking:** the Android app is now native, in Kotlin and Jetpack Compose with Material 3. The release APK keeps the app id `io.github.ilyasturki.ordo`, its signing key and its launcher name, so it updates the old app in place and home-screen icons keep working, but it starts signed out: pair it again. Edits the old app hasn't synced stay behind, so let it sync before updating. Plain http now reaches only `localhost`, `127.0.0.1` and the emulator's host, so a server on plain http in a LAN needs TLS first. Scan the code needs Google Play services; without them, paste the link
- The app's edits run through the TUI's own Go code, bound with gomobile, and it syncs with `ordo serve` over the same API, in the background too once the app is closed
- Finished tasks stay in place in the app, struck through, and its filter follows the TUI's rules, counting what it hides in each category; a tap on that count shows the hidden rows until you change project or filter
- Projects are tabs over a pager that follows the finger, one `+` opens a sheet on the keyboard to add tasks, and the task page shows only the fields that are set, with Mark done floating at the bottom right. The not-synced icon only shows once edits stay stuck, and tapping it says why. On a paired server, the device shows as `<model> · Ordo`
- Search in the app: the icon beside the filter opens a search bar on the keyboard and narrows the open project's list in place, matching what the TUI's `/` matches: titles, descriptions, separators and category names, smartcase. Back closes it
- Images in the app: a task's thumbnails open a full-screen viewer with pinch to zoom and a swipe between images, and the row of unset fields adds one from the photo picker or the camera. A photo waits on the phone with the edits, counted in the not-synced explanation, and reaches `ordo serve` later even with the app closed. Formats the server refuses, such as a phone's HEIC, become a JPEG
- Settings → Devices in the app: the paired devices with this one marked, signing one out behind a confirm, Sign out, and Pair a new device with the code, the link and the share sheet
- Passphrase sign-in in the app: the sign-in field also takes a server's address, and the passphrase shows only when that server has one, or at once when signing back in after a session ended
- The app moves like any other Android app: its pages open and close, and follow the back gesture, with the system's own animation
- The app does what the TUI does: Mark done stays on the task's page, statuses read Todo, In Progress, Completed and Cancelled (in the app's messages too), a new separator lands first in its category and needs no label, labels read as typed and can be cleared, a task or separator the filter would hide is refused with the TUI's words, finished rows keep their marks, an empty category says No tasks, the filter lists Category before Tag, a sub-category's menu adds a sibling, and deleting the open project opens the first one. "Details" is now "description"
- Undo brings back a category deleted in the app, with its sub-categories and tasks, through a new `category.restore` op in [`docs/sync.md`](docs/sync.md)
- `ordo serve pair` says to scan the code or paste the link in the Ordo app, and warns when it has no address to put in the link but localhost, which a phone can't reach
- `enter` or `space` on a category's `(N hidden by the filter)` row shows those tasks in place, marked `(filtered out)`, until the cursor leaves the category, where it used to clear the whole filter. The filter and its saved copy stay as they were; `f` → Clear all filters still clears it
- The project picker dialog (`ctrl+p`) has the rounded border of the other dialogs, so it stands out from the list behind it. The full-screen picker stays frameless
- Four tag marks instead of eight: `t`, `T`, the filter and the app offer the four coloured dots and drop the diamonds. A diamond already in a project file, or sent by an older app, reads as its colour's dot
- `just native` builds and installs the app as Ordo Dev, `just native-test` and `just native-ui <emulator>` test it, and `just dev-phone` runs it on the USB phone, signed in by itself to a throwaway seeded server

### Removed

- **Breaking:** the browser app and the old Capacitor Android app. `ordo serve` now answers only its sync API, under `/api`, and a short plain-text page anywhere else, so a browser can no longer open your projects and a device pairs through `ordo serve pair` or Devices on a paired device. Every build of `ordo serve` is now the same, with no bundle and no placeholder page. Selecting several tasks and nesting a task by dragging it went with the old app
- `just web`, `just wasm`, `just android`, `just phone` and `just web-deps-hash`. `nix build` no longer needs node

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

[Unreleased]: https://github.com/ilyasturki/ordo/compare/v0.14.2...HEAD
[0.14.2]: https://github.com/ilyasturki/ordo/compare/v0.14.1...v0.14.2
[0.14.1]: https://github.com/ilyasturki/ordo/compare/v0.14.0...v0.14.1
[0.14.0]: https://github.com/ilyasturki/ordo/compare/v0.13.0...v0.14.0
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
