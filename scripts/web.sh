#!/usr/bin/env bash
# The web app, run inside `nix develop .#web`:
#   check          typecheck, Vitest, both builds, then every Playwright spec
#   e2e [args]     both builds, then Playwright with args (a spec file, --project site-webkit-phone…)
#   build <build>  web/build/<build>, the static site of `site` or `private`
set -euo pipefail

cd "$(dirname "$0")/.."

scripts/wasm.sh build
cd web
bun install --frozen-lockfile

build() {
	ORDO_WEB="$1" bun run build
}

e2e() {
	local want have
	want=${ORDO_PLAYWRIGHT:?run this in nix develop .#web, whose browsers Playwright drives}
	have=$(node -p "require('./node_modules/@playwright/test/package.json').version")
	if [[ "$want" != "$have" ]]; then
		echo "web/package.json pins @playwright/test $have, the flake's browsers are for $want: pin $want" >&2
		exit 1
	fi
	build site
	build private
	bunx playwright test "$@"
}

case "${1:-check}" in
check)
	bun run check
	bun run test
	e2e
	;;
e2e)
	shift
	e2e "$@"
	;;
build)
	build "${2:?site or private}"
	;;
*)
	echo "usage: $0 [check|e2e [args]|build site|private]" >&2
	exit 2
	;;
esac
