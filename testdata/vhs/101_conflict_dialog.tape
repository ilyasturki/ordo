Output "/tmp/ordo-vt/vhs-out/101_conflict_dialog.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
# The CLI changes the title the editor holds open, so both sides changed one field.
Type "clear && (sleep 2 && ./ordo -p Default task edit 'Build the main dashboard' --title 'Build main dashboard (web)' -q >/dev/null 2>&1 &) ; ./ordo -p Default"
Enter
Sleep 800ms
Show

Type "gg"
Type "jj"
Enter
Sleep 2200ms
Type " v2"
Enter
Sleep 1500ms
Screenshot "/tmp/ordo-vt/vhs-out/101_dialog.png"
Sleep 500ms

# b keeps theirs and adds mine below it.
Type "b"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/101_both.png"

# u takes the choice back.
Type "u"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/101_undone.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
