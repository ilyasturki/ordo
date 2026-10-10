Output "/tmp/phas-vt/vhs-out/32_description_preview.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# Grow the dashboard description past the focused cap: 9 seeded lines + 12.
Hide
Type `clear && sed -i 's/how much history to keep"/how much history to keep'"$(for i in $(seq -w 1 12); do printf '\\\\n- note %s' $i; done)"'"/' /tmp/phas-vt/data/projects/*.json`
Enter
Sleep 300ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1000ms
Show

# Expand inline descriptions: the long dashboard description shows a 3-line
# preview with a "… +N more lines" tail while the cursor is elsewhere.
Type "zd"
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/32_preview_truncated.png"

# On the description row it grows to 10 lines, and the tail points to gi.
Type "j"
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/32_preview_focused_capped.png"
Sleep 500ms

# A match past the cap moves the window onto it.
Type "/note 11"
Sleep 600ms
Enter
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/32_preview_search_window.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
