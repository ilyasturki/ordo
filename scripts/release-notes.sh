#!/usr/bin/env bash
# Print a version's CHANGELOG.md notes; fail when they are missing or empty.
# Usage: scripts/release-notes.sh <X.Y.Z>
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <X.Y.Z>" >&2
  exit 1
fi

notes="$(awk -v heading="## [$1]" '
  index($0, heading) == 1 { found = 1; next }
  found && /^## \[/ { exit }
  found { print }
' CHANGELOG.md | sed '/./,$!d')"

if [[ -z "$notes" ]]; then
  echo "CHANGELOG.md has no notes under '## [$1]'" >&2
  exit 1
fi

printf '%s\n' "$notes"
