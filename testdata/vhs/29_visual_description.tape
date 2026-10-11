Output "/tmp/ordo-vt/vhs-out/29_visual_description.gif"
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

# Expand inline descriptions; the first task carries one.
Type "zd"
Sleep 500ms

# Visual mode from the task row: extending down lands on the description
# row, which joins the selection band.
Type "v"
Sleep 300ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/29_visual_desc_in_range.png"

# Extend one more row (next task) — the band covers task, description, task.
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/29_visual_desc_swept.png"

# Shift the block down and back up; the description travels with its task.
Type "J"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/29_visual_desc_shift_down.png"

Type "K"
Sleep 400ms
Escape
Sleep 300ms

# Visual mode anchored on the description row itself (no longer rejected):
# jump back to the top, step onto the description row, then press v.
Type "gg"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "v"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/29_visual_anchor_on_desc.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms

Hide
Type "q"
