#!/usr/bin/env bash
# Recompute this system's hash of the app's node_modules in nix/web-deps-hashes.nix.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
hashes="$repo/nix/web-deps-hashes.nix"
system="$(nix eval --impure --raw --expr builtins.currentSystem)"
fake_hash="sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

original="$(cat "$hashes")"
current="$(sed -nE "s/^  $system = \"(sha256-[^\"]*)\";\$/\1/p" "$hashes")"

write_hash() {
	local updated
	if [[ -n "$(sed -n "/^  $system = /p" "$hashes")" ]]; then
		updated="$(sed -E "s|^(  $system = )\"[^\"]*\";\$|\1\"$1\";|" "$hashes")"
	else
		updated="$(sed '$d' "$hashes")"$'\n'"  $system = \"$1\";"$'\n}'
	fi
	printf '%s\n' "$updated" > "$hashes"
}

trap 'printf "%s\n" "$original" > "$hashes"' EXIT

write_hash "$fake_hash"

echo "==> building .#web.deps for $system to compute the hash (this runs bun install)..."
build_output="$(nix build "$repo#web.deps" --no-link 2>&1 || true)"

new_hash="$(printf '%s\n' "$build_output" | sed -nE 's/.*got:[[:space:]]+(sha256-[^[:space:]]+).*/\1/p')"
if [[ -z "$new_hash" ]]; then
	echo "$build_output" >&2
	echo "error: could not parse the new hash from the build output above" >&2
	exit 1
fi

trap - EXIT
write_hash "$new_hash"

if [[ "$new_hash" == "$current" ]]; then
	echo "==> $system deps hash already up to date: $new_hash"
else
	echo "==> $system deps hash updated"
	echo "    old: ${current:-none}"
	echo "    new: $new_hash"
fi
