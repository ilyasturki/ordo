Output "/tmp/ordo-vt/vhs-out/161_next_version_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/releases.json -q && clear && ./ordo -p Releases"
Enter
Sleep 800ms
Show

# Fold every category so the versions sit one per row, then stand on v0.10.0.
Type "zc"
Sleep 300ms
Type "gg}}"
Sleep 200ms
Screenshot "/tmp/ordo-vt/vhs-out/161_start.png"
Sleep 300ms

# gn adds v0.11.0 above it, the newest side of this project, with the name open to tweak.
Type "gn"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/161_editor.png"
Sleep 300ms
Type " - plan"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/161_saved.png"
Sleep 300ms

# One u takes the whole version back.
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/161_undo.png"
Sleep 300ms

# FUTURE holds no version: the status line says so.
Type "G"
Type "gn"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/161_no_version.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
