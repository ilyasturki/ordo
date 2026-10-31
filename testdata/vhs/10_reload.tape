Output "/tmp/ordo-vt/vhs-out/10_reload.gif"
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

# r reloads by hand; an outside change reloads by itself, see 94_outside_change.
Type "r"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/10_after_reload.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
