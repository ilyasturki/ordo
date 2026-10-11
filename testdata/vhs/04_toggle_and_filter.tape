Output "/tmp/ordo-vt/vhs-out/04_toggle_and_filter.gif"
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

# Move past category header onto the first task
Type "j"
Sleep 300ms

Type " "
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/04_after_toggle.png"

Type "f"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/04_filter_hub.png"

# Drill into Status, toggle Todo, return to hub
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/04_filter_status.png"
Type " "
Sleep 300ms
Escape
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/04_filter_hub_with_status.png"

# Drill into Priority
Type "j"
Sleep 200ms
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/04_filter_priority.png"
Type " "
Sleep 300ms
Escape
Sleep 300ms

# Clear all (from Priority row, j twice -> Clear all)
Type "jj"
Sleep 200ms
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/04_filter_after_clear.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms

Hide
Type "q"
