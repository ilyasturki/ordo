Output "/tmp/ordo-vt/vhs-out/24_visual_separator.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Land on the first task, drop a labeled separator below it.
Type "j"
Sleep 200ms
Type "-"
Sleep 300ms
Enter
Sleep 200ms
Type "Milestone"
Sleep 200ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/24_separator_placed.png"

# Step up onto the task just above the separator, enter visual mode, and
# extend down across the separator so the divider is swept into the band.
Type "k"
Sleep 200ms
Type "v"
Sleep 300ms
Type "j"
Sleep 200ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/24_visual_spans_separator.png"

# Shift the whole block (separator included) down one row.
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/24_block_shifted.png"

# Delete the block; the separator goes with it. Confirm.
Type "d"
Sleep 300ms
Type "y"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/24_block_deleted.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
