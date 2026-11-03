Output "/tmp/ordo-vt/vhs-out/103_filter_toggle.gif"
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

# The status options with the marks the rows draw, Todo ticked.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/103_status_marks.png"
Sleep 300ms

# The priority options with their glyphs.
Escape
Type "j"
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/103_priority_marks.png"
Sleep 300ms
Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/103_filtered.png"
Sleep 300ms

# F turns the filter off: every task, no badge, no hidden rows.
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/103_off.png"
Sleep 300ms

# The dialog says it is off.
Type "f"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/103_off_dialog.png"
Sleep 300ms
Type "q"
Sleep 300ms

# F again brings the same filter back.
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/103_on_again.png"
Sleep 300ms

Hide
Type "q"
