Output "/tmp/ordo-vt/vhs-out/27_description_save_keys.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 1000ms
Show

# Enable inline descriptions and open the in-app editor on a description row.
Type "zd"
Sleep 400ms
Type "j"
Sleep 300ms
Enter
Sleep 500ms
# Footer should now read: enter save · shift+enter newline · esc cancel
Screenshot "/tmp/ordo-vt/vhs-out/27_editor_open_hints.png"

# Shift+Enter (delivered as Alt+Enter / ESC+CR by kitty & ghostty) inserts a
# newline instead of saving.
Type " and more"
Sleep 200ms
Alt+Enter
Sleep 200ms
Type "a second line"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/27_editor_two_lines.png"

# Plain Enter now saves and returns to the list.
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/27_saved_multiline.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
