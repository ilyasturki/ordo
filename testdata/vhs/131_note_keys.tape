Output "/tmp/ordo-vt/vhs-out/131_note_keys.gif"
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

# o adds a note below the cursor, like _.
Type "jj"
Sleep 200ms
Type "o"
Sleep 300ms
Type "Ask Sam first"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/131_o_note.png"
Sleep 300ms

# c turns the task below into a note, and back.
Type "j"
Sleep 200ms
Type "c"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/131_c_note.png"
Sleep 300ms
Type "c"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/131_c_task.png"
Sleep 300ms

# On a visual range, every row follows the first: here, all become notes.
Type "vjj"
Sleep 200ms
Type "c"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/131_visual_c.png"
Sleep 300ms

# One undo step brings the three tasks back.
Escape
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/131_undo.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
