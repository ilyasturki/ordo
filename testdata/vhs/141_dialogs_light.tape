Output "/tmp/ordo-vt/vhs-out/141_dialogs_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 800
Set FontSize 16
Set Theme "Builtin Light"
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

# The red band and the underlined keys read on a light theme.
Type "d"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/141_light_delete.png"
Sleep 300ms
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/141_light_delete_keep.png"
Sleep 300ms
Escape
Sleep 300ms

Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/141_light_move.png"
Sleep 300ms
Escape
Sleep 300ms

Ctrl+P
Sleep 500ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/141_light_group.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Escape
Sleep 300ms
Escape
Sleep 300ms
Hide
Type "q"
