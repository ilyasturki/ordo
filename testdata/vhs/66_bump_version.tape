Output "/tmp/ordo-vt/vhs-out/66_bump_version.gif"
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

# Fold every category so the versions sit one per row.
Type "zc"
Sleep 300ms
Type "gg}}}"
Sleep 200ms
Screenshot "/tmp/ordo-vt/vhs-out/66_start.png"
Sleep 300ms

# ctrl+a on "v0.9.1 - polish" adds one to the patch and keeps the label.
Ctrl+A
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_patch_up.png"
Sleep 300ms

# gm bumps the minor and zeroes the patch; gM bumps the major.
Type "gm"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_minor.png"
Sleep 300ms
Type "k"
Type "gM"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_major.png"
Sleep 300ms

# ctrl+x on v0.8.5 would take v0.8.4; ctrl+a would take v0.8.6, which exists.
Type "jjjj"
Sleep 200ms
Ctrl+A
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_clash.png"
Sleep 300ms

# A visual range of v0.8.6, v0.8.5 and v0.8.0 bumps together: v0.8.5 takes
# v0.8.6 while v0.8.6 moves on to v0.8.7. FUTURE has no version and is skipped.
Type "kvjjj"
Sleep 200ms
Ctrl+A
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_range.png"
Sleep 300ms

# One u undoes the whole range.
Escape
Sleep 300ms
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_undo.png"
Sleep 300ms

# A category without a version stays as it is.
Type "G"
Ctrl+A
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/66_no_version.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
