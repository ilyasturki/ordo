Output "/tmp/ordo-vt/vhs-out/102_field_keys.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type `clear && ./ordo task add "Fix the login form today" -C Ergonomy -q`
Enter
Sleep 800ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1000ms
Show

Type "/Fix the login"
Sleep 400ms
Enter
Sleep 400ms
Escape
Sleep 400ms
Enter
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_title_open.png"
Sleep 600ms

# Two ctrl+w in a row kill "form today" as one kill.
Ctrl+W
Sleep 200ms
Ctrl+W
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_killed.png"
Sleep 600ms

Type "page"
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_typed.png"
Sleep 600ms

# VHS can't name ctrl+_: the Type below holds its byte, 0x1f.
Type ""
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_undo_word.png"
Sleep 600ms

Ctrl+Y
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_yanked.png"
Sleep 600ms

Enter
Sleep 900ms
Screenshot "/tmp/ordo-vt/vhs-out/102_saved.png"
Sleep 900ms

Hide
Type "q"
Sleep 500ms
