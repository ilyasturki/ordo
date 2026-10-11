Output "/tmp/ordo-vt/vhs-out/51_versions.gif"
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

# Open versions show done/total and the work in progress; shipped ones fold,
# dim and stack, and v0.8.6 warns that it shipped with open tasks.
Screenshot "/tmp/ordo-vt/vhs-out/51_overview.png"
Sleep 300ms

# s on a task of v0.9.1 ships the version: it folds and joins the stack.
Type "gg}}}"
Sleep 200ms
Type "j"
Sleep 200ms
Type "s"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/51_shipped.png"
Sleep 300ms

# u reopens it.
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/51_undo.png"
Sleep 300ms

# tab unfolds a shipped version by hand; its header stays dim.
Type "G{{{{"
Sleep 200ms
Tab
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/51_unfolded.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
