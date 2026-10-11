# A on the last category of a long project: the editor and the new category stay on screen.
# 24 rows: Height 500 with Padding 10 at FontSize 16 (1 row = 20px, rows = (Height-20)/20).
Output "/tmp/ordo-vt/vhs-out/57_add_category_at_end.gif"
Set Shell "bash"
Set Width 1000
Set Height 500
Set FontSize 16
Set Padding 10
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

Type "G"
Sleep 300ms
Type "A"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/57_editor.png"
Sleep 200ms
Type "Later"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/57_created.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
