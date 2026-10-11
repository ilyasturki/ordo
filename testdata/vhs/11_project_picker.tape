Output "/tmp/ordo-vt/vhs-out/11_project_picker.gif"
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

# Open the project picker
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/11_picker_open.png"

Type "k"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/11_picker_on_add.png"

# Enter add mode and type a name to confirm the dialog width is stable
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/11_picker_add_empty.png"

Type@40ms "Side Project"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/11_picker_add_typed.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

# Cancel without creating so the seed stays intact
Escape
Sleep 300ms
Escape
Sleep 300ms

Hide
Type "q"
