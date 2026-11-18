Output "/tmp/ordo-vt/vhs-out/140_dialogs.gif"
Set Shell "bash"
Set Width 1200
Set Height 800
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/nested.json -q && ./ordo project group Versions Work -q && ./ordo project group Default Perso -q && clear && ./ordo -p Versions"
Enter
Sleep 800ms
Type "/Rest timer"
Enter
Sleep 300ms
Show

# d on a task with subtasks: Delete is the red band, Keep subtasks takes s, no hint row.
Type "d"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/140_delete.png"
Sleep 300ms

# j moves the band; esc cancels.
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/140_delete_keep.png"
Sleep 300ms
Escape
Sleep 300ms

# The move picker draws the same rows, a digit heading each.
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/140_move.png"
Sleep 300ms
Escape
Sleep 300ms

# The tag picker too, its dot keeping its colour in the band.
Type "T"
Sleep 300ms
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/140_tag.png"
Sleep 300ms
Escape
Sleep 300ms

# m in the picker lists the groups in use, the current one marked.
Ctrl+P
Sleep 500ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/140_group.png"
Sleep 300ms

# j then enter puts Versions in Perso.
Type "j"
Sleep 300ms
Enter
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/140_group_set.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Hide
Type "q"
