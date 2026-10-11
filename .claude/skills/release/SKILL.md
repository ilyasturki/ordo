---
name: release
description: Roll the changelog's Unreleased notes into a new version, then bump VERSION and tag it
argument-hint: "[major|minor|patch|X.Y.Z]"
disable-model-invocation: true
allowed-tools: Read, Edit, Skill, AskUserQuestion, Bash(git log *), Bash(git describe *), Bash(just test), Bash(just bump *)
---

# Release

`VERSION` holds the version; the flake and the Go ldflags read it. `just bump` (`scripts/bump.sh`) writes it, commits, and creates the annotated `vX.Y.Z` tag. It refuses a dirty tree, an existing tag, and a version without notes in `CHANGELOG.md`. The tag push runs `.github/workflows/release.yml`, which publishes those notes as the release body.

## Context

Current version: !`cat VERSION`

Commits since the last tag:

!`git log --oneline $(git describe --tags --abbrev=0)..HEAD`

Unreleased notes:

!`sed -n '/^## \[Unreleased\]/,/^## \[[0-9]/p' CHANGELOG.md`

## Steps

1. **Version.** Take `$ARGUMENTS`: `major`, `minor` or `patch` bumps the current version, `X.Y.Z` is used as is. With no argument, propose a level from the notes (anything under Added, Changed or Removed is a minor before 1.0, only Fixed is a patch) and confirm it with AskUserQuestion. Done when you hold the exact `X.Y.Z`.
2. **Notes.** Every user-facing commit since the last tag has a line under `## [Unreleased]`. Add the missing ones under Added, Changed, Removed or Fixed, written for a user (the key, flag or behaviour, no commit prefixes), merging commits that make one change; skip refactors, tests, style and CI. Keep any upgrade warning that opens the section.
3. **Roll.** Rename `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD` with today's date, and put an empty `## [Unreleased]` above it.
4. **Commit** `CHANGELOG.md` through the commit skill.
5. **Check.** Run `just test`. Stop and report on a failure; a stale `vendorHash` in `flake.nix` is the usual cause after a dependency change.
6. **Bump.** Run `just bump X.Y.Z` with the exact version. Stop and report if it refuses.
7. **Report** the change (`v0.9.0 -> v0.10.0`) and the push that publishes it: `git push --follow-tags`. Leave the push to the user.
