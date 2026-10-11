Output "/tmp/ordo-vt/vhs-out/47_subtasks.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/nested.json -q && clear && ./ordo -p Versions"
Enter
Sleep 800ms
Show

# a adds a task below the first one; > nests it under the task above.
Type "a"
Sleep 200ms
Type "Welcome copy"
Enter
Sleep 200ms
Type ">"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/47_nested.png"
Sleep 300ms

# a on a subtask adds a sibling subtask.
Type "a"
Sleep 200ms
Type "Pick the hero image"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/47_sibling_subtask.png"
Sleep 300ms

# < takes a subtask out, after its parent's block.
Type "<"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/47_unnested.png"
Sleep 300ms

# Deleting a parent says what happens to its subtasks; k keeps them.
Type "gg"
Type "j"
Type "j"
Sleep 200ms
Type "d"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/47_delete_parent.png"
Sleep 300ms
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/47_kept_subtasks.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
