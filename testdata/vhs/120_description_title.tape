Output "/tmp/ordo-vt/vhs-out/120_description_title.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type `clear && ./ordo task add "Rewrite the ferry timetable parser so it handles daylight saving, Sunday services, the winter schedule and every late boat the harbour adds without notice" -C Ergonomy --description "$(seq -f 'Step %g of the plan' 1 60)" -q && ./ordo task add "Call the harbour office" -C Ergonomy --description "Ask for the winter PDF" -q`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# A long description: the editor names its task on one row under the heading,
# cut by the screen's width, and takes the screen's height.
Type "zd"
Sleep 300ms
Type "/Rewrite the ferry"
Enter
Sleep 300ms
Escape
Type "j"
Sleep 300ms
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/120_tall.png"
Sleep 300ms

# A short one keeps the usual height, its top where the tall one's was.
Escape
Sleep 300ms
Type "/Call the harbour"
Enter
Sleep 300ms
Escape
Type "j"
Sleep 300ms
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/120_short.png"
Sleep 300ms

# Typed lines grow the box downward.
Alt+Enter
Type "Lines:"
Alt+Enter
Type "one"
Alt+Enter
Type "two"
Alt+Enter
Type "three"
Alt+Enter
Type "four"
Alt+Enter
Type "five"
Alt+Enter
Type "six"
Alt+Enter
Type "seven"
Alt+Enter
Type "eight"
Alt+Enter
Type "nine"
Alt+Enter
Type "ten"
Alt+Enter
Type "eleven"
Alt+Enter
Type "twelve"
Alt+Enter
Type "thirteen"
Alt+Enter
Type "fourteen"
Alt+Enter
Type "fifteen"
Alt+Enter
Type "sixteen"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/120_grown.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
