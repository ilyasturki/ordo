Output "/tmp/ordo-vt/vhs-out/50_visual_nest.gif"
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

# v selects "Fix the set counter" and "Workout history chart"; > nests both under "Rest timer".
Type "gg"
Type "jjjjjjjj"
Sleep 200ms
Type "v"
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/50_selected.png"
Sleep 300ms
Type ">"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/50_nested.png"
Sleep 300ms

# < takes the selected subtasks back out, still selected.
Type "<"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/50_unnested.png"
Sleep 300ms

# A selected sub-category refuses >, since nesting goes one level deep.
Escape
Sleep 200ms
Type "gg"
Type "jjjjjjjjjj"
Type "v"
Sleep 200ms
Type ">"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/50_refused.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
Escape
Sleep 500ms

Hide
Type "q"
