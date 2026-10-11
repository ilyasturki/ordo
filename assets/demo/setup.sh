# Sourced by the demo tapes from the repo root, before recording starts.
demo_dir=$(mktemp -d)
export ORDO_DATA_PATH="$demo_dir/data"
export ORDO_CONFIG_PATH="$demo_dir/config"
# `p` reads this file instead of the desktop clipboard.
export ORDO_CLIPBOARD_FILE="$demo_dir/clipboard"
export PATH="$demo_dir/bin:$PATH"

# A failed step must stop here: PATH would otherwise fall back to an installed ordo.
go build -o "$demo_dir/bin/ordo" ./cmd/ordo &&
    cp assets/demo/checklist.md "$ORDO_CLIPBOARD_FILE" &&
    ordo import assets/demo/homelab.json -q &&
    ordo import assets/demo/ferry.json -q &&
    cp assets/demo/state.json "$ORDO_DATA_PATH/state.json"
