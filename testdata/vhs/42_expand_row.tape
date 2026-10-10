Output "/tmp/phas-vt/vhs-out/42_expand_row.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# ze shows the focused task in full, title and description, until the cursor
# leaves it. Seed a long title, and grow the dashboard description past a
# screenful: 9 seeded lines + 40.
Hide
Type `clear && ./phasionary task add "$(for i in $(seq 1 150); do printf 'w%03d ' $i; done)END" -C Ergonomy -q`
Enter
Sleep 500ms
Type `clear && sed -i 's/how much history to keep"/how much history to keep'"$(for i in $(seq -w 1 40); do printf '\\\\n- note %s' $i; done)"'"/' /tmp/phas-vt/data/projects/*.json`
Enter
Sleep 300ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 1000ms
Show

# Land on the long title, then expand it in place.
Type "/w001"
Sleep 400ms
Enter
Sleep 400ms
Escape
Sleep 400ms
Type "ze"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/42_expand_title.png"
Sleep 400ms

# Leaving the row folds it back to three lines.
Type "j"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/42_collapsed_on_leave.png"
Sleep 400ms

# Descriptions are hidden, yet ze on the dashboard shows its whole body.
Type "gg"
Sleep 300ms
Type "jj"
Sleep 300ms
Type "ze"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/42_expand_description.png"
Sleep 400ms

# The row is taller than the screen: j scrolls through it a row at a time,
# with the cursor kept on it, before moving on.
Type "j"
Sleep 300ms
Type "jjjjjjjjjjjj"
Sleep 800ms
Screenshot "/tmp/phas-vt/vhs-out/42_expand_scrolling.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
