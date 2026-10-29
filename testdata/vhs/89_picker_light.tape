Output "/tmp/ordo-vt/vhs-out/89_picker_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && for n in Alpha Bravo; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo project link Default -q && ./ordo"
Enter
Sleep 1500ms
Show

# The dialog's border separates it from the list behind it on a light theme too.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/89_light_dialog_on_linked_row.png"

Type "l"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/89_light_dialog_unlinked.png"

Escape
Sleep 300ms
Hide
Type "q"
Sleep 300ms
Type "clear && ./ordo pick"
Enter
Sleep 1500ms
Show

# The full-screen picker stays frameless.
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/89_light_full.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Hide
Type "q"
