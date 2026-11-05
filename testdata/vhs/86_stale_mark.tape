Output "/tmp/ordo-vt/vhs-out/86_stale_mark.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# The CLI stands in for the phone: it completes a task while the TUI is open.
Hide
Type "clear && (sleep 2 && ./ordo -p Default task status 'Add user preferences panel' completed -q >/dev/null 2>&1 &) ; ./ordo -p Default"
Enter
Sleep 800ms
Show

# The title editor is open as the file changes, so no reload takes the row away.
Type "gg"
Type "jj"
Enter
Sleep 2200ms

# Committing the title merges with the CLI's edit, with no dialog.
Type "!"
Enter
Sleep 1500ms
Screenshot "/tmp/ordo-vt/vhs-out/86_merged.png"

# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
