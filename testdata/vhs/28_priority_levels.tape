# Exercises the two new priority levels (critical ⇈ / trivial ⇊) and their l/h
# keybindings. The extreme icons are single-cell arrows so every priority row's
# title aligns identically with the ▲/▼ rows. Verifies icons, colors, level
# order, and the human-readable labels.
Output "/tmp/ordo-vt/vhs-out/28_priority_levels.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Cursor starts on "Build the main dashboard" (high ▲). Pressing l once bumps
# it above high to the new top level: critical ⇈.
Type "l"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_critical.png"

# Move down to "Add fold/unfold animations" (low ▼) and press h once to drop
# it below low to the new bottom level: trivial ⇊. The list now shows every
# level at once — critical ⇈, high ▲, medium (plain), low ▼, trivial ⇊ —
# all single-cell icons so every title aligns at the same column.
Type "jjjj"
Sleep 300ms
Type "h"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_trivial.png"

# Info panel confirms the human-readable "Trivial" label.
Type "gi"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_info_trivial.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
