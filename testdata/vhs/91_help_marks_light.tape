Output "/tmp/ordo-vt/vhs-out/91_help_marks_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 1500ms
Show

Type "?"
Sleep 400ms
Type "a"
Sleep 300ms
Type "G"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/91_light_help_marks.png"

Type "/"
Sleep 200ms
Type "tag"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/91_light_help_marks_filtered.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Escape
Sleep 200ms
Hide
Type "q"
