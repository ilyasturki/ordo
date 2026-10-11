Output "/tmp/ordo-vt/vhs-out/53_move_to_version.gif"
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

# M on an inbox task lists the open versions and their areas; the lowest open
# version, the next to ship, is preselected. Shipped versions are left out.
Type "M"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/53_picker.png"
Sleep 300ms

# Enter moves it there and leaves the cursor in the inbox for the next one.
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/53_moved.png"
Sleep 300ms

# A visual range goes in one move; the picker remembers the last target.
Type "vj"
Sleep 200ms
Type "M"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/53_visual_picker.png"
Sleep 300ms
Type "j"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/53_visual_moved.png"
Sleep 300ms

# One u undoes the whole move.
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/53_undo.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
