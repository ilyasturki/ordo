Output "/tmp/ordo-vt/vhs-out/19_picker_search.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# Seed a handful of projects with overlapping names so a query narrows to
# several matches. run.sh re-seeds before every tape, so these stay isolated.
Hide
Type "clear && for n in Phaser Alpha Metaphase Billing Nginx; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo -p Default"
Enter
Sleep 1500ms
Show

# Open the project picker.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/19_picker_open.png"

# Press "/" to start filtering: the pinned "+ New Project" row is replaced by a
# "/" prompt, and the footer switches to the filter hints.
Type "/"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/19_picker_filter_empty.png"

# Type a query: the list narrows live to matching projects, the matched span in
# each name is highlighted, and a match count shows on the prompt line.
Type@60ms "pha"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/19_picker_filter_typed.png"

# Arrow down through the filtered matches (New Project stays out of reach).
Down
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/19_picker_filter_moved.png"

# Esc clears the filter and restores the full list, keeping the cursor on the
# project that was highlighted.
Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/19_picker_filter_cleared.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

# Esc again closes the picker.
Escape
Sleep 300ms

Hide
Type "q"
