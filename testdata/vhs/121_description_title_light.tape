Output "/tmp/ordo-vt/vhs-out/121_description_title_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type `clear && ./ordo task add "Rewrite the ferry timetable parser so it handles daylight saving, Sunday services, the winter schedule and every late boat the harbour adds without notice" -C Ergonomy --description "$(seq -f 'Step %g of the plan' 1 60)" -q`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# 120_description_title's tall editor on a light theme: the title row stays muted but legible.
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
Screenshot "/tmp/ordo-vt/vhs-out/121_tall_light.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
