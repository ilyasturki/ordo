Output "/tmp/ordo-vt/vhs-out/18_reverse_categories.gif"
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

# Seed order top-to-bottom: Feature, Fix, Ergonomy, Documentation, Research.
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/18_before_reverse.png"

# S flips the whole category order in one shot: Research now on top, Feature last.
Type "S"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/18_after_reverse.png"

# S is its own inverse: pressing it again restores the original order.
Type "S"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/18_toggled_back.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
