Output "/tmp/ordo-vt/vhs-out/77_filter_hidden_rows.gif"
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

# Filter on In progress: Feature keeps one task and gets "(4 hidden by the
# filter)" under it; Fix, Documentation and Research show only their row.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type "j"
Type " "
Sleep 200ms
Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/77_filtered.png"
Sleep 500ms

# j stops on the hidden row: it takes the cursor and the bar offers enter.
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/77_hidden_row_focused.png"
Sleep 500ms

# enter clears the whole filter and leaves the cursor on Feature.
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/77_after_enter.png"
Sleep 500ms

Hide
Type "q"
