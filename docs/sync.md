# Sync API

`ordo serve` answers the phone apps and the browser app under `/api`. Each keeps a copy of every project and a queue of the edits made on it, and edits the copy through `internal/operations`, compiled to WebAssembly (`cmd/ordo-core`) or, for the native Android app, bound with gomobile (`native/core`), so an edit made with no signal shows at once and reaches the files later. The server replays the queue through the same code, under the same lock as every other writer (see [file-format.md](file-format.md#locks)).

Every answer is JSON, and gzipped for a client that sends `Accept-Encoding: gzip`. An error answers `{"error": "<message for the user>", "protocol": 1}`.

## Signing in

`GET /api/hello` needs no token. It answers `{"protocol": 1, "has_passphrase": true}`, so the sign-in screen knows whether to offer the passphrase.

`POST /api/pair` takes `{"token": "<code>", "name": "<device name>"}`, where the code is the one `ordo serve pair` prints, or the link around it. It answers `{"token": "<device token>", "protocol": 1}`. The phone app sends that token on every later request as `Authorization: Bearer <device token>`, and the API answers it from any origin. With `"cookie": true` the server sets the token as an `HttpOnly` cookie and leaves it out of the answer: the browser app, served from `ordo serve` itself, signs in that way, and a request carrying the cookie must come from the same origin.

`POST /api/login` takes `{"passphrase": "<passphrase>", "name": "<device name>"}` and answers like `/api/pair`, `"cookie": true` included.

A code works once. A wrong code or passphrase counts toward the same limit as the pairing or sign-in page. A request without a valid token or cookie answers 401.

## Devices

A signed-in device manages the others:

- `GET /api/devices` answers `{"devices": [...], "has_passphrase": true}`. Each device has `id`, `name`, `created_at`, `last_seen`, `from` (the name of the device that paired it, missing when it signed in another way), `also` (the names of the devices revoking it also signs out: the ones it paired, and theirs) and `self` (true for the device asking).
- `POST /api/devices/pair` takes `{"passphrase": "<passphrase>"}`, needed when one is set, and answers `{"link": "<pairing link>", "qr": "data:image/png;base64,...", "minutes": 10}`. The link pairs one device within `minutes`, and replaces the last one this device minted.
- `POST /api/devices/<id>/revoke` signs out that device and every device it paired. It answers `{"self": true}` when the device asking is one of them.
- `POST /api/logout` signs out the device asking, and the devices it paired, and answers 204.

## `POST /api/sync`

One round trip sends the queue and brings back every project that changed.

```json
{
  "protocol": 1,
  "ops": [
    { "id": "q1", "project": "<pid>", "action": "status.set", "ids": ["<task>"], "value": "completed" },
    { "id": "q2", "project": "<pid>", "action": "task.add", "new": "<task id the client minted>", "cat": "<category>", "title": "Retry uploads" }
  ],
  "have": { "<pid>": "<stamp>" }
}
```

```json
{
  "protocol": 1,
  "results": [
    { "id": "q1", "status": "applied", "msg": "Marked completed" },
    { "id": "q2", "status": "applied", "msg": "Added “Retry uploads”" }
  ],
  "projects": [
    { "id": "<pid>", "name": "Ferry", "stamp": "<stamp>", "data": { "...": "the project, as in file-format.md" } }
  ]
}
```

- The server applies the ops in order. Consecutive ops on one project share one lock and one save, and an op that fails leaves the project as it was before it, so the ops after it still run. A project op runs on its own, see [Projects](#projects).
- `results` has one entry per op, in the same order. `status` is `applied`, `unchanged` (the file already held the edit), `conflict` (a title or details edit typed over text the file no longer holds; `theirs` has the file's text), `gone` (none of its rows, or its category or project, is left; a delete of what is already gone is `unchanged`) or `refused` (`msg` says why). Every status but `applied` leaves the file as it was, and the client drops the op from its queue whatever its status.
- `projects` lists every project. `stamp` changes whenever the file's bytes do. `data` is left out when the stamp matches the one the client sent in `have`. A project that is missing from the list was deleted. A project whose file doesn't load has `error` and no `data`, and the client keeps its copy.
- After a sync, the client takes `data` as its new copy of each project, then applies the ops it queued since it sent the request.

A client and a server with different `protocol` numbers refuse to sync: the server answers 409.

## Ops

An op names its rows by ID and says where they end up, never how far they moved, because it can be sent twice when an answer is lost, and land on a file another device changed in the meantime. Sending an op a second time changes nothing and answers `unchanged`.

| Action | Fields | Does |
| --- | --- | --- |
| `task.add` | `new`, `cat`, `title`, `parent` | Adds a task last in `cat`, or last under `parent` wherever it is. |
| `task.insert` | `new`, `cat`, `ids`, `title` | Adds a task below the block of `ids[0]`, or first in `cat`. |
| `separator.add` | `new`, `cat`, `ids`, `title` | The same, for a separator labelled `title`. |
| `task.title` | `ids`, `title`, `base` | Sets the title, if the file's title is still `base`. |
| `task.description` | `ids`, `text`, `base` | Sets the details, if the file's details are still `base`. |
| `task.attach` | `ids`, `lines` | Appends the image lines the details don't hold yet. |
| `task.delete` | `ids`, `keep` | Deletes the rows with their subtasks, or frees the subtasks with `keep`. |
| `status.set` | `ids`, `value` | `todo`, `in_progress`, `completed` or `cancelled`. |
| `priority.set` | `ids`, `value` | A priority, or empty to clear it. |
| `tag.set` | `ids`, `value` | A tag mark, or empty to clear it. A `-diamond` name from an older client lands as its colour's dot. |
| `estimate.set` | `ids`, `value` | Minutes, `0` to clear. |
| `task.place` | `ids`, `cat`, `parent`, `before` | Moves the rows' blocks above `before` in `cat`, under `parent` when set, or last without `before`. |
| `task.move` | `ids`, `cat` | Moves the rows' blocks last in `cat`. |
| `task.restore` | `cat`, `parent`, `before`, `rows` | Puts back rows a delete removed, with their IDs and every field, above `before` in `cat`, under `parent` when set, or last without `before`. `rows` is one top-level row followed by its subtasks, or one row. |
| `category.add` | `new`, `parent`, `title` | Adds a category last among the top-level ones, or last among `parent`'s sub-categories. |
| `category.rename` | `cat`, `title` | Renames the category. |
| `category.delete` | `cat` | Deletes the category with its sub-categories and all their tasks. |
| `category.restore` | `cats`, `before` | Puts back a category a delete removed, above its sibling `before`, or last among its siblings without it. `cats` is the category followed by its sub-categories, with their tasks. |
| `project.create` | `title` | Creates the project `project` names, with no categories. |
| `project.rename` | `title` | Renames the project. |
| `project.delete` | | Deletes the project. |

`new` is a client-minted ID (see [IDs](file-format.md#ids-and-timestamps)), so the ops queued after an add can name its row or category. An add whose `new` is already in the project is `unchanged`. A row an op was made beside can be gone by the time it lands: `task.insert` then adds first in `cat`, `task.add`, `task.place`, `task.restore` and `category.add` drop a missing `parent` and land top-level, and `task.place` and `task.restore` put a missing `before` last.

`task.restore` takes `rows` as the Task objects of [file-format.md](file-format.md#task) and checks them like a file it imports. It skips the rows the project holds again; the subtasks of a row that is back go last under it. A category name a sibling already has is `refused`, and so is a project name another project has.

`category.restore` takes `cats` as the Category objects of [file-format.md](file-format.md#category) and checks them like a file it imports. It skips the categories and rows the project holds again, drops a missing parent and lands top-level, and puts a missing `before` last. A name a sibling already has is `refused`.

`status.cycle`, `task.shift`, `task.nest` and `task.unnest` step from the state they were made on, so the server refuses them from a queue. The client runs them on its copy and queues the `status.set` or `task.place` that describes where they landed; `operations.Perform` returns that op.

Undoing an edit queues more ops: `operations.Undo` lists the ones that put back what an applied op changed, from the project before and after it, and the client runs and queues them like any edit.

### Projects

A project op's `project` is the project it creates, renames or deletes. `project.create` takes the ID the client minted, like `new`, so the ops queued after it can name the project. The server runs each project op on its own, between the runs of edits around it; a create or a rename holds the lock that guards project names (see [Locks](file-format.md#locks)). A create whose ID is already on disk, a rename to the name the project already has and a delete of a project already gone are `unchanged`; a rename of a project that is gone is `gone`.

## Images

The client names an image the way the store does: `<hash>.<ext>`, the first 32 hex digits of the sha256 of its bytes and the extension of its type (see [Images](file-format.md#images)). `PUT /api/attachments/<name>` with the bytes as the body stores it, and is refused when the bytes don't hash to the name. Once it is stored, a `task.attach` op adds `![<name>](attachments/<hash>.<ext>)` to the task. `GET /api/attachments/<name>` serves a stored image.

## Live changes

`GET /api/events` is a stream of server-sent events that stays open: each time a project file changes, from a sync, the TUI or Syncthing, it sends `event: change` with the project's ID as `data`. The app syncs when one arrives instead of waiting for its next poll. The stream sends a comment every 25 seconds so proxies keep it open.

## Protocol

`protocol` goes up when an older client or server would misread the other: an op whose fields change meaning, or an answer shaped differently. A new action doesn't need it: a server that doesn't know an action refuses it with a message to update.
