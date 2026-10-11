Output "/tmp/ordo-vt/vhs-out/07_open_url.gif"
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

# Add a task with two URLs in its title so gx opens the picker.
Type "a"
Sleep 200ms
Type "See https://example.com and https://example.org for refs"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/07_open_url_task_added.png"

# Open URL picker with gx (cursor lands on the new task).
Type "gx"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/07_open_url_picker.png"

# Move selection within picker.
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/07_open_url_picker_second.png"

# Cancel without opening.
Escape
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/07_open_url_closed.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
