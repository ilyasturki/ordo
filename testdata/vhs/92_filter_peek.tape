Output "/tmp/ordo-vt/vhs-out/92_filter_peek.gif"
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

# In progress filter, cursor on Feature's "(4 hidden by the filter)" row.
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
Screenshot "/tmp/ordo-vt/vhs-out/92_hidden_row.png"
Sleep 500ms

# enter shows them in place, muted and marked (filtered out).
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/92_peek.png"
Sleep 500ms

# Moving inside Feature, up to its header, keeps them.
Type "jkkk"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/92_peek_header.png"
Sleep 500ms

# Leaving Feature hides them again behind the row.
Type "}"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/92_peek_ended.png"
Sleep 500ms

Hide
Type "q"
