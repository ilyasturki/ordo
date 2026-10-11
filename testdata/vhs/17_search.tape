Output "/tmp/ordo-vt/vhs-out/17_search.gif"
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

# Open search and type a query that matches several tasks across categories.
# The prompt shows on the bottom line with a live match count; every match is
# highlighted and the cursor jumps to the first.
Type "/"
Sleep 400ms
Type "add"
Sleep 700ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_typing.png"

# Commit with Enter: highlights persist, cursor stays on the first match.
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_committed.png"

# Step forward through matches with n (jumps across categories).
Type "n"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_next.png"

Type "n"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_next2.png"

# Step back with N.
Type "N"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_prev.png"

# Esc clears the highlight (vim :nohlsearch), keeping the cursor put.
Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/17_search_cleared.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
