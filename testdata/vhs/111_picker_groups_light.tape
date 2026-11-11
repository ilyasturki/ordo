Output "/tmp/ordo-vt/vhs-out/111_picker_groups_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && for n in Ferry Garden Books Loose; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo project group Ferry Work -q && ./ordo project group Default Work -q && ./ordo project group Garden Perso -q && ./ordo project group Books perso -q && ./ordo -p Default"
Enter
Sleep 2000ms
Show

# The separators read on a light theme, in the dialog and with the field open.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/111_light_groups.png"
Sleep 500ms
Type "m"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/111_light_group_field.png"
Sleep 500ms
Escape
Sleep 300ms
Escape
Sleep 300ms
Hide
Type "q"
Sleep 300ms
Type "clear && ./ordo pick"
Enter
Sleep 1500ms
Show

# The full-screen picker keeps the groups and its footer on one row.
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/111_light_full.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Hide
Type "q"
