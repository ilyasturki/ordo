Output "/tmp/ordo-vt/vhs-out/97_notes.gif"
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

# Descriptions and a labelled separator on screen, to tell a note apart from both.
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

# _ adds a note below the cursor and opens its editor at once.
Type "_"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_note_editor.png"

# A long note wraps like a title.
Type "Ask Sam before merging the dashboard: the reports page still reads the old totals, and the export needs the same fix before Friday's release."
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_note.png"

# It moves like a separator, and never nests.
Type "J"
Sleep 300ms
Type ">"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_note_moved.png"

# It deletes at once, like a separator.
Type "d"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_note_deleted.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
