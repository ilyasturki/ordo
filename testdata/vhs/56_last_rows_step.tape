# j onto the row before a 1-row last item (here the second-to-last task) from a scrolled view must show it.
# 24 rows: Height 500 with Padding 10 at FontSize 16 (1 row = 20px, rows = (Height-20)/20).
Output "/tmp/ordo-vt/vhs-out/56_last_rows_step.gif"
Set Shell "bash"
Set Width 1000
Set Height 500
Set FontSize 16
Set Padding 10
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# gg, then 20 x j lands on "Investigate performance bottlenecks", one row above the last task.
Type "gg"
Type@40ms "jjjjjjjjjjjjjjjjjjjj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/56_second_to_last.png"
Sleep 200ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/56_last.png"
Sleep 200ms

# The project line is off screen, so status messages sit on "↑ more above".
Type "x"
Sleep 200ms
Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/56_status_scrolled.png"
Sleep 200ms
Type "vk"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/56_visual_scrolled.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
