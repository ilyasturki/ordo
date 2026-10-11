Output "/tmp/ordo-vt/vhs-out/06_reorder_across_categories.gif"
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

# Land on the last task of the first category ("Feature").
# Initial selection is the first task; jump forward to Fix, then back one.
Type "}"
Sleep 200ms
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/06_before_J_into_next.png"

# J at the bottom of "Feature" should move task to top of "Fix"
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/06_after_J_into_next.png"

# K at the top of "Fix" should move task back to bottom of "Feature"
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/06_after_K_back.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
