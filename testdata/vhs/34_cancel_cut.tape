Output "/tmp/phas-vt/vhs-out/34_cancel_cut.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"
Env PHASIONARY_CLIPBOARD_FILE "/tmp/phas-vt/clipboard"

# x writes the clipboard, as y does; here that is the file run.sh points at.
Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

# Cut a single task: the status line says p moves it and esc cancels, and the
# shortcut bar keeps saying so. Then esc: the ghosting and the ✂ badge go away
# and the status line reports the cancellation.
Type "j"
Sleep 200ms
Type "x"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/34_task_marked_cut.png"

Escape
Sleep 1s
Screenshot "/tmp/phas-vt/vhs-out/34_task_cut_cancelled.png"

# Same for a visual range — every row in it must undim at once.
Type "v"
Sleep 300ms
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms
Type "x"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/34_range_marked_cut.png"

Escape
Sleep 1s
Screenshot "/tmp/phas-vt/vhs-out/34_range_cut_cancelled.png"

# Search wins the first esc: cut, search, esc drops only the highlight and the
# cut survives; the second esc cancels it.
Type "x"
Sleep 300ms
Type "/"
Sleep 200ms
Type "e"
Sleep 300ms
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/34_cut_then_search.png"

Escape
Sleep 1s
Screenshot "/tmp/phas-vt/vhs-out/34_esc_keeps_cut.png"

Escape
Sleep 1s
Screenshot "/tmp/phas-vt/vhs-out/34_second_esc_cancels.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
