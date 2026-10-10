Output "/tmp/phas-vt/vhs-out/43_tag_marks_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "clear && ./phasionary -p Default"
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
Screenshot "/tmp/phas-vt/vhs-out/43_light_marks_rows.png"
Sleep 300ms

# A diamond on the selection bar becomes a reverse block like a dot does.
Type "k"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/43_light_diamond_selected.png"
Sleep 300ms

Type "T"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/43_light_picker.png"
Sleep 300ms
Escape
Sleep 200ms

# The tag filter lists each mark on its own row.
Type "f"
Sleep 300ms
Type "jjj"
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/43_light_filter_tags.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
Escape
Escape
Sleep 200ms

Hide
Type "q"
