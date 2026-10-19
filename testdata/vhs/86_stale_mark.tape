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
Sleep 3000ms
Show

# The TUI's next edit is refused, and the header says what r would drop.
Type "gg"
Type "jj"
Type "l"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/86_refused.png"

# The next key clears the status line, not the mark; it counts every refused edit.
Type "j"
Type "h"
Type "j"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/86_mark_stays.png"

# Scrolled down, the "more above" line carries the mark.
Type "G"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/86_mark_scrolled.png"

# r drops them and shows the CLI's edit.
Type "gg"
Type "r"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/86_reloaded.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
