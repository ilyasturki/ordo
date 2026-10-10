# File format

Phasionary 0.9.0 carried its own sync, made of a change journal, a SQLite server and a web app. Each change to the data model had to be taught to all three. The server was never deployed, and Syncthing already carried the files between machines, so the whole stack was removed. Phasionary stays a terminal tool. Any other client, a phone app included, lives in its own repository and works on the files described here.

## Layout

```
~/.local/share/phasionary/       or $PHASIONARY_DATA_PATH
├── projects/
│   ├── <id>.json                one project per file
│   ├── <id>.json.tmp            a write in progress
│   ├── <id>.json.lock           flock for that project
│   └── .global.lock             flock for creating and renaming projects
└── state.json                   this device's UI state
```

Only `projects/` travels between devices. `state.json` holds folds, cursor positions, the project order and which directory opens which project; it belongs to one device and stays there. The config file, `~/.config/phasionary/config.json`, is per device too.

A file in `projects/` is a project only when its name is `<id>.json`, `<id>` is a valid ID, and the `id` inside the file is that same `<id>`. Phasionary ignores every other file in the directory, from the `.tmp` and `.lock` sidecars to renamed and conflict copies.

## Project JSON

```json
{
  "schema": 1,
  "id": "872e2818-9fd7-44c6-e33e-e6ef4c934ef0",
  "name": "Default",
  "created_at": "2026-05-23T16:37:34Z",
  "updated_at": "2026-10-07T09:12:03Z",
  "categories": [
    {
      "id": "de2e32f7-b882-b2cb-309a-9048dbf3936c",
      "name": "Feature",
      "created_at": "2026-05-23T16:37:34Z",
      "tasks": [
        {
          "id": "9652ac64-1187-6d0e-3f4c-5a2b7d8e9f01",
          "title": "Build the main dashboard",
          "status": "in_progress",
          "created_at": "2026-05-23T16:37:34Z",
          "updated_at": "2026-06-02T18:40:11Z",
          "priority": "high",
          "estimate_minutes": 480
        }
      ]
    }
  ]
}
```

Fields marked optional are left out of the file when empty or zero, and a reader treats a missing one as empty or zero.

### Project

| Field | Type | Rule |
| --- | --- | --- |
| `schema` | integer | Format version, see [Schema](#schema). Missing means 1. |
| `id` | string | Same as the filename. |
| `name` | string | One line. Unique across projects, compared trimmed and case-insensitive. |
| `created_at` | timestamp | |
| `updated_at` | timestamp | Restamped on every save. |
| `categories` | array | In display order. |

### Category

| Field | Type | Rule |
| --- | --- | --- |
| `id` | string | |
| `name` | string | One line. Unique within the project, compared trimmed and case-insensitive. |
| `created_at` | timestamp | |
| `updated_at` | timestamp | Optional. |
| `estimate_minutes` | integer | Optional. Set on the category itself, not summed from its tasks. |
| `tasks` | array | In display order, separators included. |

### Task

| Field | Type | Rule |
| --- | --- | --- |
| `id` | string | |
| `title` | string | One line. On a separator, its label, often empty. |
| `status` | string | `todo`, `in_progress`, `completed` or `cancelled`. Empty on a separator. |
| `created_at` | timestamp | |
| `updated_at` | timestamp | Restamped whenever the task changes. |
| `priority` | string | Optional. `critical`, `high`, `medium`, `low` or `trivial`. |
| `completion_date` | timestamp | Optional. Set when the status becomes `completed`, cleared when it leaves it. |
| `estimate_minutes` | integer | Optional. |
| `description` | string | Optional. The one field that may hold newlines and tabs. |
| `kind` | string | Optional. `separator` turns the row into a divider; missing means an ordinary task. |
| `tag_color` | string | Optional. `green`, `blue`, `magenta` or `cyan`. |
| `tag_label` | string | Optional. Only alongside a `tag_color`. |

Every string is valid UTF-8 with no control characters: nothing below U+0020, no U+007F, nothing from U+0080 to U+009F. `description` is the exception for newline and tab. Phasionary strips control characters from any file it loads, since a terminal would run them as commands.

## IDs and timestamps

An ID is 1 to 64 characters from `A-Z`, `a-z`, `0-9`, `-` and `_`. Phasionary makes them from 16 random bytes printed as lowercase hex in 8-4-4-4-12 groups, the same for projects, categories and tasks. An ID never changes and is never reused, even after the thing it named is deleted.

Timestamps are RFC 3339 in UTC to the second: `2026-05-23T16:37:34Z`.

## Schema

`schema` is the version of this format, 1 today.

- A reader refuses a file whose `schema` is higher than the one it knows. Loading it and saving it back would drop whatever the newer version added.
- A writer stamps the version it writes, on every save.
- A file without the field reads as version 1.

The number goes up when an older phasionary would misread a file, or would lose data by saving it without a field it does not know (see below). A new field whose loss is harmless needs no bump.

## Unknown fields are dropped

Phasionary decodes a project into fixed types and writes back only the fields listed above. A field another app adds survives until phasionary next saves that project, then it is gone. Another client writes these fields and no others. A new field lands in phasionary first.

## Writing a file

1. Write the whole project to `<id>.json.tmp` in the same directory.
2. fsync it.
3. Rename it over `<id>.json`.
4. fsync the directory.

A reader then sees either the old file or the new one in full. Syncthing should ignore the temp files: `(?d)*.tmp` in `.stignore`.

## Locks

Phasionary holds a flock on `<id>.json.lock` while it reads, changes and writes a project, and on `.global.lock` while it creates or renames one, since that is where it checks names for duplicates. These are advisory locks inside one machine and protect nothing across devices. Two devices that edit the same project between syncs end in a Syncthing conflict, locks or not.

A client on the same machine as phasionary takes the same locks. A client on another device has no use for them. Keep `*.lock` in `.stignore`, or the lock files replicate as empty files.

## Edits from outside

The TUI keeps the open project in memory and saves snapshots of it. When it loads the file it notes its identity (inode, size, modification time), and checks it again before every save. If Syncthing, the CLI or another app has changed the file in between, the save is refused and the status line reads "Project changed on disk; press r to reload (unsaved edits are dropped)". `r` loads the file again and drops the edits made since the last save, so the other writer's change always wins.

The CLI does each command as one read, change and write under the lock, so it always starts from the current file. Another client should do the same: re-read before writing, or check that the file is still the one it loaded.

## Conflict copies

When two devices change the same file before Syncthing reconciles them, Syncthing keeps one version as `<id>.json` and renames the other to `<id>.sync-conflict-<date>-<time>-<device>.json`. Phasionary never loads, edits or deletes a conflict copy. When a project with one opens, the TUI names the file in its status line. Merging the two and deleting the copy is a manual job.
