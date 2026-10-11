Output "/tmp/ordo-vt/vhs-out/10_reload.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
# Schedule an external edit to land ~2s after ordo starts.
Type `(sleep 2 && f=$(ls /tmp/ordo-vt/data/projects/*.json) && sed -i 's/Build the main dashboard/Externally edited title/' "$f") >/dev/null 2>&1 &`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Initial in-memory view, before the external write fires.
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/10_before_external_edit.png"

# Wait for the backgrounded sed to mutate the JSON file on disk.
Sleep 1800ms
Screenshot "/tmp/ordo-vt/vhs-out/10_stale_in_memory.png"

# Press r — reload from disk, status flashes on the project line.
Type "r"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/10_after_reload.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
