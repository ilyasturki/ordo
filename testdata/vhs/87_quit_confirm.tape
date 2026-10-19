Output "/tmp/ordo-vt/vhs-out/87_quit_confirm.gif"
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

Type "gg"
Type "jj"
Type "l"
Sleep 300ms
Type "j"
Sleep 300ms

# q asks before dropping the refused edit.
Type "q"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/87_quit_asks.png"

# n keeps the session; ctrl+c asks the same.
Type "n"
Sleep 300ms
Ctrl+C
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/87_ctrl_c_asks.png"

# y quits, and ordo says on stderr what was lost, exiting non-zero.
Type "y"
Sleep 800ms
Type "echo exit=$?"
Enter
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/87_quit_reported.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
