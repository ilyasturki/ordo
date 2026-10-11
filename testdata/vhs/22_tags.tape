Output "/tmp/ordo-vt/vhs-out/22_tags.gif"
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

# Cursor starts on the first task. `t` cycles the color; the dot stays its
# own color even though the row is focused (reverse).
Type "t"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/22_tag_green_focused.png"

# `T` opens the mark picker, seeded on the current mark.
Type "T"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/22_editor_open.png"

# Choose magenta.
Type "jj"
Sleep 200ms
Screenshot "/tmp/ordo-vt/vhs-out/22_editor_color.png"
Sleep 300ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/22_row_tagged_focused.png"

# Move the cursor off the task so the dot shows on an unfocused row too.
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/22_row_tagged_unfocused.png"

# Info view still lists the tag.
Type "gi"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/22_tag_info.png"
Type "q"
Sleep 200ms

# Remove the tag via the picker: back onto the task, T, up to "None".
Type "k"
Sleep 200ms
Type "T"
Sleep 200ms
Type "kkk"
Sleep 200ms
Screenshot "/tmp/ordo-vt/vhs-out/22_editor_none.png"
Sleep 300ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/22_row_removed.png"

# Bulk: visual-select two tasks and open the shared editor.
Type "v"
Sleep 150ms
Type "j"
Sleep 150ms
Type "T"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/22_bulk_editor.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
Escape
Sleep 200ms

Hide
Type "q"
