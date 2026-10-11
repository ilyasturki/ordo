# Sourced by the demo tapes from the repo root, before recording starts.
demo_dir=$(mktemp -d)
export PHASIONARY_DATA_PATH="$demo_dir/data"
export PHASIONARY_CONFIG_PATH="$demo_dir/config"
# `p` reads this file instead of the desktop clipboard.
export PHASIONARY_CLIPBOARD_FILE="$demo_dir/clipboard"
export PATH="$demo_dir/bin:$PATH"

# A failed step must stop here: PATH would otherwise fall back to an installed phasionary.
go build -o "$demo_dir/bin/phasionary" ./cmd/phasionary &&
    cp assets/demo/checklist.md "$PHASIONARY_CLIPBOARD_FILE" &&
    phasionary import assets/demo/homelab.json -q &&
    phasionary import assets/demo/ferry.json -q
