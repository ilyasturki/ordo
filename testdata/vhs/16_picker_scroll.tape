Output "/tmp/ordo-vt/vhs-out/16_picker_scroll.gif"
Set Shell "bash"
Set Width 1200
Set Height 480
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# Seed enough projects that the list must scroll, then launch on Default. The
# run.sh harness re-seeds before every tape, so these extra projects stay
# isolated to this run.
Hide
Type "clear && for n in Alpha Bravo Charlie Delta Echo Foxtrot Golf Hotel India Juliet Kilo Lima Mike November Oscar; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo -p Default"
Enter
Sleep 2500ms
Show

Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/16_picker_open.png"

# ctrl+d pages down half a screen; the pinned New Project stays put while the
# project list scrolls (note the "more above" indicator).
Ctrl+D
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/16_picker_scrolled.png"

# The pinned New Project affordance is reached by going to the first project
# (g) and stepping up past it (k).
Type "g"
Sleep 200ms
Type "k"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/16_picker_new_project.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Sleep 300ms
Type "q"
