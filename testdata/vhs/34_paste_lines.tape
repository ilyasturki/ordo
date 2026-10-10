Output "/tmp/phas-vt/vhs-out/34_paste_lines.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"
Env PHASIONARY_CLIPBOARD_FILE "/tmp/phas-vt/clipboard"

# Seeds the clipboard file, which `p` reads. VHS cannot send a bracketed paste,
# so the terminal-paste routes are covered by paste_chooser_test.go instead.
Hide
Type "clear && printf '## Sprint 12\nDesign the API\n- [ ] write handlers\n- [~] review\n- [x] write tests\nShip it\n' > /tmp/phas-vt/clipboard"
Enter
Sleep 500ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

Type "j"
Sleep 300ms
Screenshot "/tmp/phas-vt/vhs-out/34_before_paste.png"

# Several lines ask how to land; the title counts what the parse found.
Type "p"
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/34_paste_chooser.png"

# Enter keeps the default, one task per line: the bare lines sit beside
# bulleted ones, so they become the separators that group them, and [~] keeps
# its in-progress status.
Enter
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/34_pasted.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
