Output "/tmp/ordo-vt/vhs-out/130_visual_move_filtered.gif"
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

# In progress filter, Feature's hidden rows shown with enter.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type "j"
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "j"
Enter
Sleep 300ms

# Four of them selected.
Type "vjjj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/130_selected.png"
Sleep 300ms

# J at Feature's end moves them into Fix: still shown, still selected.
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/130_moved.png"
Sleep 300ms

# Out of visual mode, leaving them hides them behind Fix's row.
Escape
Sleep 200ms
Type "}"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/130_left.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
