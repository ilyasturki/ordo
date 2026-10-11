Output "/tmp/ordo-vt/vhs-out/55_changelog.gif"
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

Type "gc"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/55_changelog_open.png"

Ctrl+D
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/55_changelog_scrolled.png"

Type "G"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/55_changelog_bottom.png"

Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/55_changelog_closed.png"
Sleep 500ms

Hide
Type "q"
