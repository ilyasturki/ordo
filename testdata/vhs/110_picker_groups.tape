Output "/tmp/ordo-vt/vhs-out/110_picker_groups.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && for n in Ferry Garden Books Loose; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo project group Ferry Work -q && ./ordo project group Default Work -q && ./ordo project group Garden Perso -q && ./ordo -p Default"
Enter
Sleep 2000ms
Show

# Each group under its name, the projects without one last under a bare rule.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/110_groups.png"

# m on an ungrouped project: the field offers the most recent group; P then tab completes Perso.
Type "G"
Sleep 300ms
Type "m"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/110_group_field_empty.png"
Type "P"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/110_group_field_hint.png"
Tab
Sleep 300ms
Enter
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/110_group_set.png"

# The search still runs across groups.
Type "/o"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/110_group_search.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Escape
Sleep 300ms
Hide
Type "q"
