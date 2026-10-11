Output "/tmp/ordo-vt/vhs-out/60_folded_runs.gif"
Set Shell "bash"
Set Width 1050
Set Height 708
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/folded_shelf.json -q && ./ordo import testdata/vhs/folded_history.json -q && clear && ./ordo -p Shelf"
Enter
Sleep 800ms
# Fold everything, then let search open the categories that hold work.
Type "zc"
Sleep 200ms
Type "/Polish"
Enter
Sleep 200ms
Type "/Rate limit"
Enter
Sleep 200ms
Type "/Launch"
Enter
Sleep 200ms
Type "gg"
Sleep 400ms
Show

# A run of 16 folded versions at the top: one row each, and the open work below them.
Screenshot "/tmp/ordo-vt/vhs-out/60_shelf_top.png"
Sleep 300ms

Type "jjjjj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/60_shelf_cursor.png"
Sleep 300ms

# Visual mode over the run is one solid band.
Type "v"
Sleep 200ms
Type "jjjj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/60_shelf_visual.png"
Sleep 300ms
Escape
Sleep 200ms

# tab unfolds one member, which splits the run.
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/60_shelf_unfold_one.png"
Sleep 300ms

# Folded sub-categories stack at their indent under an open parent.
Type "G"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/60_shelf_subcategories.png"
Sleep 300ms

Hide
Type "q"
Sleep 300ms
Type "clear && ./ordo -p History"
Enter
Sleep 800ms
Type "zc"
Sleep 200ms
Type "/Rework"
Enter
Sleep 200ms
Show

# The open version and all 19 folded versions after it fit on one screen.
Type "gg"
Sleep 300ms
Type "G"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/60_history_bottom.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
