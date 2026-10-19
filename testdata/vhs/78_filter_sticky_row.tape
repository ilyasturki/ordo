Output "/tmp/ordo-vt/vhs-out/78_filter_sticky_row.gif"
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

# Filter on Todo, then step onto "Add user preferences panel".
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "j"
Sleep 300ms

# space no longer matches the filter, yet the row stays, muted and marked.
Type " "
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/78_in_progress_kept.png"
Sleep 500ms

# space again keeps cycling the same task.
Type " "
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/78_completed_kept.png"
Sleep 500ms

# Moving off drops it: Feature now counts it among its hidden tasks.
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/78_after_j.png"
Sleep 500ms

Hide
Type "q"
