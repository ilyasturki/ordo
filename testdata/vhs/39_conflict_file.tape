Output "/tmp/ordo-vt/vhs-out/39_conflict_file.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
# Leave a Syncthing conflict copy next to the seeded project.
Type `f=$(ls /tmp/ordo-vt/data/projects/*.json) && cp "$f" "${f%.json}.sync-conflict-20260921-203208-ICE5FOT.json"`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# The launch names the conflict file on the project line.
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/39_conflict_at_launch.png"

# The notice is said once: the first keypress clears it.
Type "j"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/39_conflict_cleared.png"
Sleep 500ms

# Only one Default in the picker: the conflict copy is not a second project.
Ctrl+P
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/39_conflict_picker.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
