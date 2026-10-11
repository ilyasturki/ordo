Output "/tmp/ordo-vt/vhs-out/02_help.gif"
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

Type "?"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_card.png"

Type "jj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_card_focus.png"

# `a` switches to the full reference for this session.
Type "a"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_full.png"

Ctrl+D
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_scrolled.png"

# G reaches the very last row.
Type "G"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_bottom.png"

Type "?"
Sleep 400ms
Type "?"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_reopens_expanded.png"

# Back to the card, then run a shortcut straight from it.
Type "a"
Sleep 400ms
Type "j"
Sleep 200ms
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_enter_dispatches.png"
Sleep 500ms

# Expand again, then relaunch: a new session opens on the card.
Type "?"
Sleep 400ms
Type "a"
Sleep 400ms
Escape
Sleep 300ms
Type "q"
Sleep 300ms
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Type "?"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/02_help_fresh_launch_card.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
