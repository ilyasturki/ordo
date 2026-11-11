Output "/tmp/ordo-vt/vhs-out/108_desc_mark_light.gif"
Set Shell "bash"
Set Width 1000
Set Height 560
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/desc_mark.json -q && clear && ./ordo -p Marks"
Enter
Sleep 800ms
Show

# Descriptions hidden: a bold blue ¶ after each described title, then the
# description's first line faint in the room the row has left. The wrapped
# title carries it on its last row, next to ▣ and the estimate.
Type "g"
Sleep 200ms
Type "g"
Sleep 200ms
Type "jj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_short_selected.png"
Sleep 300ms

Type "jj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_long_selected.png"
Sleep 300ms

# A visual band over described rows, and a cut row, keep the mark inside the bar.
Type "k"
Sleep 300ms
Type "k"
Sleep 300ms
Type "v"
Sleep 200ms
Type "j"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_visual.png"
Sleep 300ms
Escape
Sleep 300ms
Type "k"
Sleep 300ms
Type "x"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_cut.png"
Sleep 300ms
Escape
Sleep 300ms

# ze shows the row's description, so that row drops its mark; the others keep it.
Type "g"
Sleep 200ms
Type "g"
Sleep 200ms
Type "jj"
Sleep 300ms
Type "ze"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_ze_row.png"
Sleep 300ms

# zd shows every description, so no row has a mark.
Type "zd"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/108_light_marks_all_shown.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
