Output "/tmp/ordo-vt/vhs-out/03_add_task.gif"
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

# Adds under the currently-focused category
Type "a"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/03_add_prompt.png"

Type@40ms "Visual regression test pass"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/03_add_typed.png"

Enter
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/03_add_saved.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
