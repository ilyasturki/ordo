Output "/tmp/ordo-vt/vhs-out/76_picker_link.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && for n in Alpha Bravo; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo project link Default -q && ./ordo"
Enter
Sleep 1500ms
Show

Ctrl+P
Sleep 500ms
Type "g"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/76_picker_before_link.png"

Type "l"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/76_picker_linked_status.png"

Hide
Type "q"
Sleep 300ms
Type "clear && ./ordo"
Enter
Sleep 1500ms
Show

Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/76_picker_reopened.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Hide
Type "q"
