Output "/tmp/ordo-vt/vhs-out/88_picker_unlink.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && for n in Alpha Bravo; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo project link Default -q && ./ordo"
Enter
Sleep 1500ms
Show

# The picker opens on the current row, which is also the linked one: l unlinks.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/88_dialog_on_linked_row.png"

Type "l"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/88_dialog_unlinked.png"

# Picking a project afterwards leaves the directory unlinked.
Type "k"
Sleep 300ms
Enter
Sleep 500ms
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/88_dialog_pick_keeps_unlinked.png"

Escape
Sleep 300ms
Hide
Type "q"
Sleep 300ms
Type "clear && ./ordo project link Bravo -q && ./ordo pick"
Enter
Sleep 1500ms
Show

# Full screen: the cursor starts on the current, linked Bravo.
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/88_full_on_linked_row.png"

Type "l"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/88_full_unlinked.png"

Type "k"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/88_full_link_hint.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Hide
Type "q"
