Output "/tmp/ordo-vt/vhs-out/12_undo_redo.gif"
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

# Move onto the second task and bump its priority
Type "j"
Sleep 300ms
Type "l"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_priority.png"

# Reorder it down with J
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_move.png"

# Undo the move
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_undo_move.png"

# Undo the priority change
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_undo_priority.png"

# Redo the priority change
Ctrl+r
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_redo_priority.png"

# Redo the move
Ctrl+r
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_redo_move.png"

# Add a new task and then undo to remove it
Type "a"
Sleep 400ms
Type@40ms "Brand new undoable task"
Sleep 300ms
Enter
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_add.png"
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_undo_add.png"

# Empty undo case
Type "uuuuuuuuuu"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/12_after_many_undos.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
