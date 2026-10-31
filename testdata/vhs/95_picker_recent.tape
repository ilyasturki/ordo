Output "/tmp/ordo-vt/vhs-out/95_picker_recent.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo project add Alpha -q >/dev/null 2>&1; sleep 1.1; ./ordo project add Bravo -q >/dev/null 2>&1; ./ordo -p Default"
Enter
Sleep 2500ms
Show

# Default was opened last, Bravo was edited after Alpha. Go to Alpha, the last row.
Ctrl+P
Sleep 500ms
Type "jjj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/95_picker_before.png"
Sleep 500ms

Enter
Sleep 500ms

# The project just opened leads the list, the one left before it follows.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/95_picker_alpha_first.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms

Hide
Type "q"
