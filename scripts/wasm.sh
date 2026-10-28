#!/usr/bin/env bash
# Build internal/core to app/static/ordo-core.wasm, beside the wasm_exec.js of the same Go.
set -euo pipefail

cd "$(dirname "$0")/.."

GOOS=js GOARCH=wasm go build -o app/static/ordo-core.wasm ./cmd/ordo-core
# Not cp: GOROOT's copy is read-only, so the next run could not overwrite it.
install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" app/static/wasm_exec.js
