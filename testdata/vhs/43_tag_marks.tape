Output "/tmp/ordo-vt/vhs-out/43_tag_marks.gif"
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

# Give eight tasks the eight marks through the picker (j steps past "None").
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
Type "j"
Type "T"
Type "jjjjj"
Enter
# Step over the Fix header.
Type "jj"
Type "T"
Type "jjjjjj"
Enter
Type "j"
Type "T"
Type "jjjjjjj"
Enter
Type "j"
Type "T"
Type "jjjjjjjj"
Enter
Sleep 800ms
Show

# All eight marks on unselected rows; the "Fix date formatting" task is
# completed, so its mark is dimmed.
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_marks_rows.png"
Sleep 300ms

# A diamond on the selection bar becomes a reverse block like a dot does.
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_diamond_selected.png"
Sleep 300ms

Type "T"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_picker.png"
Sleep 300ms
Escape
Sleep 200ms

# The tag filter lists each mark on its own row.
Type "f"
Sleep 300ms
Type "jjj"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/43_filter_tags.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
Escape
Escape
Sleep 200ms

Hide
Type "q"
