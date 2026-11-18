Output "/tmp/ordo-vt/vhs-out/180_shift_above_hidden_section.gif"
Set Shell "bash"
Set Width 1200
Set Height 600
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/hidden_section.json -q && clear && ./ordo -p Errands"
Enter
Sleep 800ms
Show

# Todo filter: the "done" section's rows are all completed, so it hides whole.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "gg"
Type "jjjj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/180_filtered.png"
Sleep 300ms

# K on the vet visit lands it under "Post the parcel", above the hidden section: no "done" appears.
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/180_shifted.png"
Sleep 300ms

# Clearing the filter shows it above "done".
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/180_unfiltered.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
