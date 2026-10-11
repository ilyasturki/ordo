Output "/tmp/phas-vt/vhs-out/54_next_version.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "./phasionary import testdata/vhs/releases.json -q && clear && ./phasionary -p Releases"
Enter
Sleep 800ms
Show

# A on a version proposes the patch after the newest one and, in this
# newest-first project, opens it above the newest version, below the inbox.
Type "gg}}}j"
Sleep 200ms
Type "A"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/54_proposal.png"
Sleep 300ms

# Enter keeps the proposal.
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/54_created.png"
Sleep 300ms

# The name stays editable: A again, then a minor bump instead.
Type "A"
Sleep 200ms
Ctrl+U
Type "v0.11.0"
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/54_edited.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
