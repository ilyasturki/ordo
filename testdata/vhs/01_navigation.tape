Output "/tmp/ordo-vt/vhs-out/01_navigation.gif"
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

Type "jjjjj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/01_after_jjjjj.png"

Type "G"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/01_after_G.png"

Type "gg"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/01_after_gg.png"

Type "}"
Sleep 300ms
Type "}"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/01_after_next_category.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
