Output "/tmp/phas-vt/vhs-out/32_long_title.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# Titles have no length limit. This tape covers what bounds one instead: the
# list clamps the row to ui.MaxLineRows, a search moves that window onto the
# match, and the info dialog scrolls so the whole title stays readable
# somewhere. The inline editor is 38_multiline_title_edit.
Hide
Type `clear && ./phasionary task add "$(for i in $(seq 1 150); do printf 'w%03d ' $i; done)END" -C Ergonomy -q`
Enter
Sleep 500ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1000ms
Show

Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/32_long_title_clamped.png"
Sleep 400ms

# The cursor on it changes nothing: a focused title keeps the clamp, so moving
# past it never shifts the rows below (ze expands it: 42_expand_row).
Type "/w001"
Sleep 400ms
Enter
Sleep 400ms
Escape
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/32_long_title_focused_clamped.png"
Sleep 400ms

# A match past the clamp still shows: the window opens on it.
Type "/w120"
Sleep 600ms
Enter
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/32_long_title_search_window.png"

Escape
Sleep 400ms

# The info dialog holds the title in full, and scrolls.
Type "gi"
Sleep 900ms
Screenshot "/tmp/phas-vt/vhs-out/32_long_title_info_top.png"

# A no-op keystroke, as above: the window is already at the top, so this
# redraws it unchanged and flushes the queued screenshot on the top state
# instead of on the frame G draws.
Type "k"
Sleep 400ms

Type "G"
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/32_long_title_info_bottom.png"

Type "q"
Sleep 600ms

Hide
Type "q"
Sleep 500ms
