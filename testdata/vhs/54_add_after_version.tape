Output "/tmp/ordo-vt/vhs-out/54_add_after_version.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "./ordo import testdata/vhs/releases.json -q && clear && ./ordo -p Releases"
Enter
Sleep 800ms
Show

# A on a task of a version opens an empty category right after that version's
# block, its areas included, like on any other category.
Type "gg}}}j"
Sleep 200ms
Type "A"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/54_editor.png"
Sleep 300ms

Type "v0.9.2"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/54_created.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
