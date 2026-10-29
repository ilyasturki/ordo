Output "/tmp/ordo-vt/vhs-out/90_help_marks.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 1500ms
Show

# The full reference ends with the legend of row marks.
Type "?"
Sleep 400ms
Type "a"
Sleep 300ms
Type "G"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/90_help_marks.png"

# The help filter finds a mark by what it means.
Type "/"
Sleep 200ms
Type "image"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/90_help_marks_filtered.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Escape
Sleep 200ms
Hide
Type "q"
