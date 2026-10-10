Output "/tmp/phas-vt/vhs-out/30_cursor_memory.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

# Park the cursor well away from the first task, where a fresh session lands.
Type "jjjjj"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/30_before_quit.png"
Sleep 500ms

# Quit and relaunch: the cursor must come back to the same row rather than
# resetting to the first task.
Hide
Type "q"
Sleep 600ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1s
Show

Screenshot "/tmp/phas-vt/vhs-out/30_after_restart.png"

# Fold the category the cursor sits in (which parks the cursor on its header),
# then restart again: fold state and cursor must both come back together.
Type "za"
Sleep 400ms
Hide
Type "q"
Sleep 600ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1s
Show

Screenshot "/tmp/phas-vt/vhs-out/30_after_restart_folded.png"
Sleep 500ms

Hide
Type "q"
