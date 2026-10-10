Output "/tmp/phas-vt/vhs-out/31_cursor_center.gif"
Set Shell "bash"
Set Width 1200
Set Height 360
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

# Short terminal, so the list overflows. Stepping down pins the cursor to the
# bottom edge — ensure-visible scrolls the minimum needed to reveal it.
Type "gg"
Sleep 300ms
Type "jjjjjjjjjj"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/31_before_quit_at_edge.png"
Sleep 500ms

# Reopening has no record of where the view sat, so the restored row should come
# back centered rather than pinned to the bottom edge it was saved from.
Hide
Type "q"
Sleep 600ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1s
Show

Screenshot "/tmp/phas-vt/vhs-out/31_after_restart_centered.png"
Sleep 500ms

Hide
Type "q"
