Output "/tmp/ordo-vt/vhs-out/33_separator_cut.gif"
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
Show

# Land on the first task and drop a labeled separator below it.
Type "j"
Sleep 200ms
Type "-"
Sleep 300ms
Enter
Sleep 200ms
Type "Milestone"
Sleep 200ms
Enter
Sleep 400ms

# Sweep task + separator + task into a visual range and mark it for cut: the
# separator row must carry the same ghosted style and ✂ badge as the tasks.
Type "k"
Sleep 200ms
Type "v"
Sleep 300ms
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms
Type "x"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/33_separator_marked_cut.png"

# Paste the block at the end of the category; the separator travels with it.
Type "G"
Sleep 200ms
Type "p"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/33_block_pasted.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
