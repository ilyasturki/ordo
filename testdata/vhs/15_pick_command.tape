Output "/tmp/ordo-vt/vhs-out/15_pick_command.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && for n in Alpha Bravo Charlie Delta; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo pick"
Enter
Sleep 2500ms
Show
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/15_pick_opens_picker.png"

Type "g"
Sleep 300ms
Type "d"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/15_pick_delete_confirm.png"

Type "y"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/15_pick_deleted.png"
Sleep 500ms

# ctrl+c on the list quits straight back to the shell.
Ctrl+C
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/15_pick_ctrl_c_quits.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
