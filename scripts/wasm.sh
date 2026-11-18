#!/usr/bin/env bash
# `build` puts internal/core for the web app in web/src/lib/core/wasm, beside the wasm_exec.js of the
# same Go; `test` vets it and runs the Go tests of what it compiles under js/wasm.
set -euo pipefail

cd "$(dirname "$0")/.."

case "${1:-build}" in
build)
	out=web/src/lib/core/wasm
	mkdir -p "$out"
	GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o "$out/ordo.wasm" ./cmd/ordo-wasm
	# Not cp: GOROOT's copy is read-only, so the next run could not overwrite it.
	install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/wasm_exec.js"
	;;
test)
	GOOS=js GOARCH=wasm go vet ./cmd/ordo-wasm ./internal/core ./internal/operations ./internal/domain
	# The js runner refuses a large environment: "total length of command line and environment variables exceeds limit".
	env -i PATH="$(go env GOROOT)/lib/wasm:$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOMODCACHE="$(go env GOMODCACHE)" \
		GOTOOLCHAIN=local GOOS=js GOARCH=wasm go test ./internal/core ./internal/operations ./internal/domain
	;;
*)
	echo "usage: $0 [build|test]" >&2
	exit 2
	;;
esac
