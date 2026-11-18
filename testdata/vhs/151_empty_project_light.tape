Output "/tmp/ordo-vt/vhs-out/151_empty_project_light.gif"
Set Shell "bash"
Set Width 1000
Set Height 560
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo project add Fresh -q && clear && ./ordo -p Fresh"
Enter
Sleep 800ms
Show

# A new project starts empty: one centred line says what to press.
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/151_empty_hint.png"
Sleep 300ms

# a in an empty project opens the first category's name, as A does.
Type "a"
Sleep 400ms
Type "Inbox"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/151_empty_first_category.png"
Sleep 300ms
Enter
Sleep 400ms

# Then a adds a task in it, and the hint is gone.
Type "a"
Sleep 400ms
Type "First task"
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/151_empty_first_task.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
