Output "/tmp/ordo-vt/vhs-out/08_description.gif"
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

Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_indicator.png"

Sleep 200ms
Type "gi"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_info.png"

Type "q"
Sleep 300ms

Type "zd"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_inline_on.png"

Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_focused.png"

Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_inapp_editor.png"

Up
Up
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_editor_cursor_up.png"

Escape
Sleep 400ms

Type "zd"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/08_description_inline_off.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
