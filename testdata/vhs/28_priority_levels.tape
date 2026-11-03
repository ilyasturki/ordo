# Exercises the five priority levels and their l/h keybindings: critical ⇈,
# high ▲, medium (no mark, the missing field), low ▼, trivial ⇊. The extreme
# icons are single-cell arrows so every priority row's title aligns identically
# with the ▲/▼ rows. Verifies icons, colors, level order, stepping through
# medium, and the labels in the info panel and the priority filter.
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
Escape
Sleep 300ms

# Up to "Add user preferences panel", seeded as the legacy "medium": the info
# panel reads "Medium". l raises it to high ▲, h brings it back to plain medium.
Type "kkk"
Sleep 300ms
Type "gi"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_info_medium.png"
Escape
Sleep 300ms
Type "l"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_medium_up.png"
Type "h"
Sleep 400ms
Type "gi"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_info_back_to_medium.png"
Escape
Sleep 300ms

# The priority filter lists five rows, Medium between High and Low, no None.
Type "f"
Sleep 300ms
Type "j"
Sleep 200ms
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/28_filter_priority.png"
Escape
Sleep 200ms
Escape
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
