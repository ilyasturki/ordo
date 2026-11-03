Output "/tmp/ordo-vt/vhs-out/99_subcategory_keys.gif"
Set Shell "bash"
Set Width 1200
Set Height 1000
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/nested.json -q && clear && ./ordo -p Versions"
Enter
Sleep 800ms
Show

# gA on a task of v0.1.13 opens the editor on its new last sub-category.
Type "/Export workouts"
Enter
Sleep 200ms
Type "gA"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/99_ga_editor.png"
Sleep 300ms
Type "api"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/99_ga_added.png"
Sleep 300ms

# K moves it among its siblings, then past the first one into v0.2.0, last there.
Type "K"
Sleep 200ms
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/99_k_across.png"
Sleep 300ms

# J takes it back down, first under v0.1.13.
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/99_j_across.png"
Sleep 300ms

# Under the first top-level category, K at the top refuses as before.
Type "gg"
Type "}"
Type "}"
Sleep 200ms
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/99_k_edge.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
