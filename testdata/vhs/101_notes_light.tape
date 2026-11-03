Output "/tmp/ordo-vt/vhs-out/101_notes_light.gif"
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

# The note of 100_notes on a light theme, beside a description and a labelled separator.
Type "zd"
Sleep 300ms
Type "j"
Sleep 200ms
Type "-"
Sleep 200ms
Enter
Type "Next up"
Enter
Sleep 300ms
Type "_"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/101_note_editor_light.png"
Type "Ask Sam before merging the dashboard: the reports page still reads the old totals, and the export needs the same fix before Friday's release."
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/101_note_light.png"
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/101_note_unselected_light.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
