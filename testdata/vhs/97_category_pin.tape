Output "/tmp/ordo-vt/vhs-out/97_category_pin.gif"
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

# gp on the last category pins it: it goes first and carries a faint ↑.
Type "G"
Type "{"
Sleep 200ms
Type "gp"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_pinned_first.png"
Sleep 300ms

# A second pin ends the pinned run; the cursor follows the category.
Type "}"
Type "}"
Sleep 200ms
Type "gp"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_pinned_second.png"
Sleep 300ms

# J at the end of the pinned run refuses: an unpinned one never passes above.
Type "J"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_pinned_edge.png"
Sleep 300ms

# gp again unpins it, first among the unpinned ones.
Type "gp"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/97_unpinned.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
