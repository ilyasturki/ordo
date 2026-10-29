Output "/tmp/ordo-vt/vhs-out/43_tag_marks_light.gif"
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

# Give four tasks the four marks through the picker (j steps past "None"),
# then the completed "Fix date formatting" task the first one.
Type "T"
Type "j"
Enter
Type "j"
Type "T"
Type "jj"
Enter
Type "j"
Type "T"
Type "jjj"
Enter
Type "j"
Type "T"
Type "jjjj"
Enter
# Step over the Fix header.
Type "jjjj"
Type "T"
Type "j"
Enter
Sleep 800ms
Show

# All four marks on unselected rows; the "Fix date formatting" task is
# completed, so its mark is dimmed.
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_light_marks_rows.png"
Sleep 300ms

# A mark on the selection bar becomes a reverse block.
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_light_mark_selected.png"
Sleep 300ms

Type "T"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_light_picker.png"
Sleep 300ms
Escape
Sleep 200ms

# The tag filter lists each mark on its own row.
Type "f"
Sleep 300ms
Type "jjj"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_light_filter_tags.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
Escape
Escape
Sleep 200ms

Hide
Type "q"
