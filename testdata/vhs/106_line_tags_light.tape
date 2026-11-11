Output "/tmp/ordo-vt/vhs-out/106_line_tags_light.gif"
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

# The tagged separator and note of 105_line_tags on a light theme.
Type "j"
Sleep 200ms
Type "-"
Sleep 200ms
Enter
Type "Next up"
Enter
Sleep 300ms
Type "t"
Sleep 300ms
Type "_"
Sleep 300ms
Type "Check the old totals first"
Enter
Sleep 300ms
Type "T"
Sleep 300ms
Type "jj"
Enter
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/106_tagged_note_light.png"
Sleep 300ms
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/106_tagged_separator_light.png"
Sleep 500ms

# A todo filter: the separator and its note stay with the todo tasks of their section.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 200ms
Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/106_status_filter_light.png"
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
