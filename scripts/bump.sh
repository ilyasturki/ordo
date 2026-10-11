#!/usr/bin/env bash
# Bump the project version: updates VERSION, commits, and creates a git tag.
#
# Usage:
#   scripts/bump.sh <X.Y.Z>     explicit semver, e.g. 1.1.0
#   scripts/bump.sh patch       1.0.0 -> 1.0.1
#   scripts/bump.sh minor       1.0.0 -> 1.1.0
#   scripts/bump.sh major       1.0.0 -> 2.0.0
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <X.Y.Z|patch|minor|major>" >&2
  exit 1
fi

if [[ ! -f VERSION ]]; then
  echo "VERSION file not found at repo root" >&2
  exit 1
fi

current="$(cat VERSION)"
if [[ ! "$current" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Current VERSION is not valid semver: '$current'" >&2
  exit 1
fi

arg="$1"
case "$arg" in
  major|minor|patch)
    IFS='.' read -r major minor patch <<< "$current"
    case "$arg" in
      major) major=$((major + 1)); minor=0; patch=0 ;;
      minor) minor=$((minor + 1)); patch=0 ;;
      patch) patch=$((patch + 1)) ;;
    esac
    new="${major}.${minor}.${patch}"
    ;;
  *)
    if [[ ! "$arg" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "Invalid version: '$arg' (expected X.Y.Z or major|minor|patch)" >&2
      exit 1
    fi
    new="$arg"
    ;;
esac

if [[ "$new" == "$current" ]]; then
  echo "Version is already $new" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree is dirty; commit or stash changes before bumping" >&2
  exit 1
fi

if git rev-parse "v${new}" >/dev/null 2>&1; then
  echo "Tag v${new} already exists" >&2
  exit 1
fi

if ! scripts/release-notes.sh "$new" >/dev/null; then
  echo "Move the Unreleased notes under '## [${new}] - $(date +%F)' and commit first" >&2
  exit 1
fi

echo "Bumping ${current} -> ${new}"
echo "${new}" > VERSION
git add VERSION
git commit -m "chore: bump version to ${new}"
git tag -a "v${new}" -m "Release v${new}"

echo
echo "Created commit and tag v${new}."
echo "Push with: git push --follow-tags"
