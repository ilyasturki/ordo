Output "/tmp/ordo-vt/vhs-out/105_path_suggest.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "rm -rf /tmp/ordo-vt/tree && mkdir -p /tmp/ordo-vt/tree/notes /tmp/ordo-vt/tree/nixos && touch /tmp/ordo-vt/tree/notes/ideas.md /tmp/ordo-vt/tree/notes/todo.md /tmp/ordo-vt/tree/news.txt /tmp/ordo-vt/tree/.env"
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# A word starting like a path opens the matching entries under the field, directories marked.
Type "a"
Sleep 200ms
Type "read /tmp/ordo-vt/tree/"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_open.png"
Sleep 200ms

# Typing on filters; down picks the next one.
Type "n"
Sleep 300ms
Down
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/105_filtered.png"
Sleep 200ms

# tab completes a directory with / and the list follows into it.
Down
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_into_dir.png"
Sleep 200ms

# Hidden files only once the name starts with a dot.
Backspace 6
Type "."
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_hidden.png"
Sleep 200ms

# esc closes the list and keeps the edit; a second esc cancels it.
Escape
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/105_dismissed.png"
Sleep 200ms
Escape
Sleep 300ms

# The description editor takes the same list.
Type "zd"
Sleep 300ms
Type "/Build the main"
Enter
Sleep 200ms
Type "j"
Sleep 200ms
Enter
Sleep 400ms
Down 10
Ctrl+E
Alt+Enter
Type "see /tmp/ordo-vt/tree/notes/"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_description.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Escape
Type "q"
