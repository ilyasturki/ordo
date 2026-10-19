Output "/tmp/ordo-vt/vhs-out/82_visual_folded_run.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
# Fold every category, then open Feature alone and land on its last task.
Type "zc"
Sleep 200ms
Type "gg"
Type "j"
Sleep 200ms
Tab
Sleep 200ms
Type "jjjjj"
Sleep 400ms
Show

Screenshot "/tmp/ordo-vt/vhs-out/82_last_open_task.png"
Sleep 300ms

# Visual j steps onto the folded header below, which stands for its tasks.
Type "v"
Sleep 200ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/82_visual_onto_folded.png"
Sleep 300ms

# G reaches the last folded header; nothing unfolds on the way.
Type "G"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/82_visual_G.png"
Sleep 300ms

# Y copies the folded categories' tasks with the visible one.
Type "Y"
Sleep 400ms
Type "q"
Sleep 400ms
Type "clear && cat /tmp/ordo-vt/clipboard"
Enter
Sleep 700ms
Screenshot "/tmp/ordo-vt/vhs-out/82_clipboard_in_shell.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
