Output "/tmp/ordo-vt/vhs-out/48_fold_nested.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/nested.json -q && clear && ./ordo -p Versions"
Enter
Sleep 800ms
Show

# tab on a subtask folds its parent, and the cursor stays on the parent.
Type "jjjj"
Sleep 200ms
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_parent_folded.png"
Sleep 300ms

# tab on an area's header folds the area alone.
Type "k"
Sleep 200ms
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_area_folded.png"
Sleep 300ms

# tab on a version folds it with its areas.
Type "{"
Sleep 200ms
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_version_folded.png"
Sleep 300ms

# Search reaches into every fold: the version, the area and the parent open.
Type "/"
Sleep 200ms
Type "sound"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_search_reveals.png"
Sleep 300ms

# zc folds every category, zo unfolds everything.
Type "zc"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_all_folded.png"
Sleep 300ms
Type "zo"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/48_all_open.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
