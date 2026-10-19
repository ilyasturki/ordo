Output "/tmp/ordo-vt/vhs-out/80_filter_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Todo filter, a task cycled out of it, and the hidden rows, on a light theme.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "j"
Type " "
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/80_kept_row_light.png"
Sleep 500ms

# The cursor on Feature's hidden row.
Type "jjjj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/80_hidden_row_light.png"
Sleep 500ms

Hide
Type "q"
