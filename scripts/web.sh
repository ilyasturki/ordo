#!/usr/bin/env bash
# Build the browser app into internal/web/spa, the bundle `ordo serve` embeds.
set -euo pipefail

cd "$(dirname "$0")/.."

scripts/wasm.sh
(cd app && bun install --frozen-lockfile && bun run build)

find internal/web/spa -mindepth 1 -maxdepth 1 ! -name placeholder.html -exec rm -rf {} +
cp -R app/build/. internal/web/spa/
