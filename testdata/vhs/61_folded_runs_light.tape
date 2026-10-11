Output "/tmp/ordo-vt/vhs-out/61_folded_runs_light.gif"
Set Shell "bash"
Set Width 1050
Set Height 708
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/folded_shelf.json -q && clear && ./ordo -p Shelf"
Enter
Sleep 800ms
Type "zc"
Sleep 200ms
Type "/Polish"
Enter
Sleep 200ms
Type "/Rate limit"
Enter
Sleep 200ms
Type "gg"
Sleep 400ms
Show

# The stacked run and its visual band stay legible on a light theme.
Type "jj"
Sleep 200ms
Type "v"
Sleep 200ms
Type "jjj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/61_shelf_visual_light.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
