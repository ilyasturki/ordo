Output "/tmp/phas-vt/vhs-out/40_cut_clipboard.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# x writes the SYSTEM clipboard as y does, so a cut task pastes into another
# app; the shell plays that app here. Quitting leaves the task where it was.
# Your clipboard's text is saved first and put back at the end.
Hide
Type "clear && timeout 2 wl-paste -n > /tmp/phas-vt/clip.bak 2>/dev/null; ./phasionary -p Default"
Enter
Sleep 800ms
Show

Type "x"
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/40_cut.png"

Type "q"
Sleep 400ms
Type "clear && timeout 2 wl-paste -n"
Enter
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/40_clipboard_in_shell.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "clear && ([ -s /tmp/phas-vt/clip.bak ] && timeout 2 wl-copy < /tmp/phas-vt/clip.bak || timeout 2 wl-copy --clear)"
Enter
Sleep 300ms
