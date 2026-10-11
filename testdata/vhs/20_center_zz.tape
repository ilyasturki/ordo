Output "/tmp/ordo-vt/vhs-out/20_center_zz.gif"
Set Shell "bash"
Set Width 1200
Set Height 360
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Jump to top, then step down into the middle of the (overflowing) list. The
# selection ends up pinned near the bottom edge by ensure-visible scrolling.
Type "gg"
Sleep 300ms
Type "jjjjjjjjjj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/20_before_zz.png"

# zz should re-center the selection vertically in the viewport.
Type "z"
Sleep 150ms
Type "z"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/20_after_zz.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
