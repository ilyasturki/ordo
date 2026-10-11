Output "/tmp/ordo-vt/vhs-out/65_move_to_category.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# m on the first Feature task lists every other category and preselects Fix,
# the one right after it.
Type "gg"
Type "jj"
Sleep 200ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_picker.png"
Sleep 300ms

# / filters the list; enter moves the task to the end of Documentation and
# leaves the cursor in Feature.
Type "/doc"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_filtered.png"
Sleep 300ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_moved.png"
Sleep 300ms

# A visual range goes in one move, and the picker remembers Documentation.
Type "vj"
Sleep 200ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_visual_picker.png"
Sleep 300ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_visual_moved.png"
Sleep 300ms

# One u undoes the whole range move.
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_undo.png"
Sleep 300ms

# m on a category header asks for a task.
Type "gg"
Type "j"
Sleep 200ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/65_on_header.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
