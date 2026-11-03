Output "/tmp/ordo-vt/vhs-out/104_filter_toggle_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
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
Screenshot "/tmp/ordo-vt/vhs-out/104_status_marks_light.png"
Sleep 300ms

# The priority options with their glyphs.
Escape
Type "j"
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/104_priority_marks_light.png"
Sleep 300ms
Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/104_filtered_light.png"
Sleep 300ms

# F turns the filter off: every task, no badge, no hidden rows.
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/104_off_light.png"
Sleep 300ms

# The dialog says it is off.
Type "f"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/104_off_dialog_light.png"
Sleep 300ms
Type "q"
Sleep 300ms

# F again brings the same filter back.
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/104_on_again_light.png"
Sleep 300ms

Hide
Type "q"
