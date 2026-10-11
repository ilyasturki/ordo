Output "/tmp/ordo-vt/vhs-out/38_multiline_title_edit.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# VHS writes a queued Screenshot on the next rendered frame, so every capture
# below is followed by a Sleep: without one the next keystroke's frame lands.
#
# The editor sizes its row from the live buffer, dropping the list's
# ui.MaxLineRows clamp. Up/down walk the wrapped rows, and a buffer taller than
# the terminal scrolls a row at a time.
Hide
Type `clear && ./ordo task add "$(for i in $(seq 1 150); do printf 'w%03d ' $i; done)END" -C Ergonomy -q && ./ordo task add "$(for i in $(seq 1 900); do printf 'x%03d ' $i; done)TAIL" -C Ergonomy -q`
Enter
Sleep 800ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1000ms
Show

# Search jumps the cursor onto the long title, then Enter opens the editor: the
# row grows from the clamp to the whole title, caret at the end.
Type "/w001"
Sleep 600ms
Enter
Sleep 600ms
Escape
Sleep 600ms
Enter
Sleep 1200ms
Screenshot "/tmp/ordo-vt/vhs-out/38_edit_wrapped.png"
Sleep 900ms

Up
Sleep 300ms
Up
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/38_caret_up_two_rows.png"
Sleep 900ms

# Typing reflows the block under the caret and pushes the rows below down.
Type "INSERTED "
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/38_typing_reflows.png"
Sleep 900ms

Escape
Sleep 500ms

# The next title is taller than the terminal: the editor opens scrolled to the
# caret at its end, with "more above" marking the rows off screen.
Type "/x001"
Sleep 600ms
Enter
Sleep 600ms
Escape
Sleep 600ms
Enter
Sleep 1500ms
Screenshot "/tmp/ordo-vt/vhs-out/38_taller_than_screen_end.png"
Sleep 900ms

# The window must hold still under a caret that only moved a row.
Up
Sleep 300ms
Up
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/38_taller_than_screen_caret_up.png"
Sleep 900ms

Ctrl+A
Sleep 1200ms
Screenshot "/tmp/ordo-vt/vhs-out/38_taller_than_screen_start.png"

Sleep 900ms
Escape
Sleep 400ms

Hide
Type "q"
Sleep 500ms
