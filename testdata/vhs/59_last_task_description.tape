# Descriptions expanded: the wrapped last task, with a 1-row description under it, is shown whole.
# 24 rows: Height 500 with Padding 10 at FontSize 16 (1 row = 20px, rows = (Height-20)/20).
Output "/tmp/ordo-vt/vhs-out/59_last_task_description.gif"
Set Shell "bash"
Set Width 1000
Set Height 500
Set FontSize 16
Set Padding 10
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type `echo '{"expand_descriptions_by_default": true}' > /tmp/ordo-vt/cfg/config.json`
Enter
Type `./ordo -p Default -q task edit 'Investigate performance bottlenecks' --title 'Investigate performance bottlenecks in the render loop, the layout cache and every place that rebuilds positions on each key press'`
Enter
Type `./ordo -p Default -q task edit 'Compare bubbletea vs tview perf' --title 'Compare bubbletea vs tview perf on a project with several hundred tasks, long dictated titles and expanded descriptions' --description 'One line of notes under the last task.'`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# The two seeded descriptions are rows too: 23 x j lands on the last task's title.
Type "gg"
Type@40ms "jjjjjjjjjjjjjjjjjjjjjjj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/59_last_title.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
