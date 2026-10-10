Output "/tmp/phas-vt/vhs-out/49_block_moves.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "./phasionary import testdata/vhs/nested.json -q && clear && ./phasionary -p Versions"
Enter
Sleep 800ms
Show

# J on a parent moves it with its subtasks, past the next task.
Type "jj"
Sleep 200ms
Type "J"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/49_parent_down.png"
Sleep 300ms
Type "K"
Sleep 200ms

# A subtask moves among its siblings and stops at the last one.
Type "j"
Sleep 200ms
Type "J"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/49_subtask_down.png"
Sleep 300ms
Type "JJ"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/49_subtask_edge.png"
Sleep 300ms

# J on a version moves it with its areas past the next version.
Type "gg"
Type "j"
Sleep 200ms
Type "J"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/49_version_down.png"
Sleep 300ms

# x on a parent cuts it with its subtasks; p moves the whole block.
Type "u"
Sleep 200ms
Type "gg"
Type "jjjj"
Sleep 200ms
Type "x"
Sleep 300ms
Type "G"
Sleep 200ms
Type "p"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/49_cut_paste_block.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
