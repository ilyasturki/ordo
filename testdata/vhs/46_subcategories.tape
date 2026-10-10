Output "/tmp/phas-vt/vhs-out/46_subcategories.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "./phasionary import testdata/vhs/nested.json -q && clear && ./phasionary -p Versions"
Enter
Sleep 800ms
Show

# Versions with their areas: sub-categories sit a level in, and a parent
# counts the finished tasks of its areas too.
Screenshot "/tmp/phas-vt/vhs-out/46_overview.png"
Sleep 300ms

# > on the last top-level category nests it under the one above.
Type "G"
Type "k"
Sleep 200ms
Type ">"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/46_nested_under_v0113.png"
Sleep 300ms

# < takes it back out, after its parent's areas.
Type "<"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/46_back_out.png"
Sleep 300ms

# A on an area adds a sibling area under the same version.
Type "gg"
Type "}}"
Sleep 200ms
Type "A"
Sleep 200ms
Type "settings"
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/46_added_area.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
