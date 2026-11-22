Output "/tmp/ordo-vt/vhs-out/190_hidden_section_line.gif"
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

# Todo filter: every row of the "done" section is completed, so it stays as one faint line.
Type "f"
Sleep 300ms
Enter
Sleep 200ms
Type " "
Sleep 200ms
Type "q"
Sleep 300ms
Type "gg"
Type "jjj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/190_line.png"
Sleep 300ms

# enter shows its rows under it, and the line offers to hide them again.
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/190_revealed.png"
Sleep 300ms
Type "k"
Enter
Sleep 400ms

# K on the vet visit, just under "next": it lands at the end of "done", which shows again.
Type "jj"
Sleep 200ms
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/190_k_into_section.png"
Sleep 300ms

# K again: above the section, which folds back into its line.
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/190_k_above.png"
Sleep 300ms

# Clearing the filter shows the vet visit above "done".
Type "F"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/190_unfiltered.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
