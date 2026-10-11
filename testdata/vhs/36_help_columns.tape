Output "/tmp/ordo-vt/vhs-out/36_help_columns.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

Type "?"
Sleep 600ms

Type "l"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/36_help_col_right.png"

# `delete` sits on the blank between the two left groups, so `h` lands on
# `search` above it — not `move up / down`, where the cursor started.
Type "jjj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/36_help_col_right_bottom.png"
Type "h"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/36_help_col_back.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "?"
Sleep 200ms
Type "q"
