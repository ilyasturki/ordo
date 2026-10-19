Output "/tmp/ordo-vt/vhs-out/79_filter_memory.gif"
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

# Filter on In progress and park the cursor on Feature's hidden row.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type "j"
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/79_before_quit.png"
Sleep 500ms

# Quit and relaunch: the filter comes back, and the cursor with it, on a row
# only the filter makes.
Hide
Type "q"
Sleep 600ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1s
Show

Screenshot "/tmp/ordo-vt/vhs-out/79_after_restart.png"
Sleep 500ms

# Clear it from the hidden row, then restart again: it stays cleared.
Enter
Sleep 300ms
Hide
Type "q"
Sleep 600ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1s
Show

Screenshot "/tmp/ordo-vt/vhs-out/79_after_clear_restart.png"
Sleep 500ms

Hide
Type "q"
