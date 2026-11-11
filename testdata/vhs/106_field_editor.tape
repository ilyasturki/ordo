Output "/tmp/ordo-vt/vhs-out/106_field_editor.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env VISUAL "/tmp/ordo-vt/fake-editor"

# ctrl+g hands the open field's unsaved text to $VISUAL and takes back what it
# wrote. The stand-in editor adds a blank line and a second line.
Hide
Type `clear && printf '#!/bin/sh\nprintf "%%s\\n\\nfrom the editor" "$(cat "$1")" > "$1"\n' > /tmp/ordo-vt/fake-editor && chmod +x /tmp/ordo-vt/fake-editor`
Enter
Sleep 300ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1000ms
Show

# A title: its lines come back joined, the field still open.
Enter
Sleep 400ms
Type " draft"
Sleep 400ms
Ctrl+g
Sleep 1000ms
Screenshot "/tmp/ordo-vt/vhs-out/106_title_back_from_editor.png"
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/106_title_saved.png"

# A description: the editor's lines stay lines, and esc still cancels.
Type "zd"
Sleep 400ms
Type "j"
Sleep 400ms
Enter
Sleep 600ms
Ctrl+g
Sleep 1000ms
Screenshot "/tmp/ordo-vt/vhs-out/106_description_back_from_editor.png"
Escape
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/106_description_cancelled.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
