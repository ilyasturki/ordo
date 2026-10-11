Output "/tmp/ordo-vt/vhs-out/09_visual_select.gif"
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

# Start visual mode on the first task and extend down 2 rows
Type "v"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_visual_enter.png"

Type "j"
Sleep 200ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_visual_range.png"

# Shift the whole range down with J, then back up with K so the
# rest of the tape sees the original task order.
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_visual_shift_down.png"

Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_visual_shift_up.png"

# Swap anchor and cursor — cursor jumps to the other end of the range
# (the cursor row is rendered with reverse colors so it stands out from
# the yellow visual range band).
Type "o"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_visual_swap.png"

# Swap back to the original cursor end before cutting.
Type "o"
Sleep 200ms

# Cut the range, then move down into the "Fix" category and paste
Type "x"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_after_cut.png"

Type "}"
Sleep 300ms
Type "p"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/09_after_paste.png"

# Now yank-then-paste: visual select two tasks, y to yank, move, paste.
# Sources should remain in place, with copies inserted at the new cursor.
Type "v"
Sleep 200ms
Type "j"
Sleep 200ms
Type "y"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_after_yank.png"

Type "}"
Sleep 200ms
Type "p"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/09_after_yank_paste.png"

# Visual delete: select two tasks, press d, confirm.
Type "v"
Sleep 200ms
Type "j"
Sleep 200ms
Type "d"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_delete_confirm.png"

Type "y"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/09_after_delete.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
