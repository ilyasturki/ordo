Output "/tmp/phas-vt/vhs-out/40_cut_clipboard.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"
Env PHASIONARY_CLIPBOARD_FILE "/tmp/phas-vt/clipboard"

# x writes the clipboard as y does, so a cut task pastes into another app; the
# shell plays that app here, reading the clipboard file. Quitting leaves the
# task where it was.
Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

Type "x"
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/40_cut.png"

Type "q"
Sleep 400ms
Type "clear && cat /tmp/phas-vt/clipboard"
Enter
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/40_clipboard_in_shell.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
