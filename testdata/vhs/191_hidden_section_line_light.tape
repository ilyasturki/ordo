Output "/tmp/ordo-vt/vhs-out/191_hidden_section_line_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 600
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/hidden_section.json -q && clear && ./ordo -p Errands"
Enter
Sleep 800ms
Show

# Todo filter: the "done" section stays as one faint line.
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
Screenshot "/tmp/ordo-vt/vhs-out/191_line.png"
Sleep 300ms

Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/191_revealed.png"
Sleep 300ms
Type "k"
Enter
Sleep 400ms

# K on the vet visit lands it at the end of "done".
Type "jj"
Sleep 200ms
Type "K"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/191_k_into_section.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
