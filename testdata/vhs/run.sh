#!/usr/bin/env bash
# Run VHS tapes against an isolated Phasionary data dir.
#
# Usage: ./testdata/vhs/run.sh [tape-name ...]
#   ./testdata/vhs/run.sh                  # run all
#   ./testdata/vhs/run.sh 03_add_task      # run one
#
# Writing tapes: always put a Sleep after the LAST Screenshot, before the
# closing Hide/q. VHS writes a screenshot on the next rendered frame, so a tape
# that quits straight after its final Screenshot can exit before that frame
# exists and silently drop the file — no error, just a missing PNG. Measured on
# 20_center_zz: 2/5 runs captured without the trailing Sleep, 8/8 with it.
set -euo pipefail

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "run.sh must be invoked from within the phasionary git repo" >&2
    exit 1
}
cd "$root"

DATA="/tmp/phas-vt/data"
CFG="/tmp/phas-vt/cfg"
OUT="/tmp/phas-vt/vhs-out"

rm -rf "$OUT" && mkdir -p "$OUT"
go build -o phasionary ./cmd/phasionary

mapfile -t TAPES < <(
    if [ "$#" -gt 0 ]; then
        for arg in "$@"; do
            name=$(basename "${arg%.tape}")
            echo "testdata/vhs/${name}.tape"
        done
    else
        find testdata/vhs -maxdepth 1 -name '*.tape' | sort
    fi
)

# Re-seed a pristine data dir before every tape so state never bleeds between
# them. Import is cheap, and an unconditional reset can't go stale the way a
# hand-maintained "these tapes mutate" list does.
seed() {
    if ! BIN="$(pwd)/phasionary" ./testdata/vhs/seed.sh "$DATA" "$CFG" >/dev/null; then
        echo "seed failed for $DATA; aborting" >&2
        exit 1
    fi
}

for tape in "${TAPES[@]}"; do
    echo
    echo "=== $(basename "$tape") ==="
    seed
    vhs "$tape"
done

echo
echo "Screenshots in $OUT (delete after verification):"
find "$OUT" -name '*.png' | sort
