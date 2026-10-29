Output "/tmp/ordo-vt/vhs-out/93_filter_peek_light.gif"
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

# The peek of 92_filter_peek on a light theme.
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
Screenshot "/tmp/ordo-vt/vhs-out/93_hidden_row_light.png"
Sleep 500ms

# enter shows them in place, muted and marked (filtered out).
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/93_peek_light.png"
Sleep 500ms

# Moving inside Feature, up to its header, keeps them.
Type "jkkk"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/93_peek_header_light.png"
Sleep 500ms

# Leaving Feature hides them again behind the row.
Type "}"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/93_peek_ended_light.png"
Sleep 500ms

Hide
Type "q"
