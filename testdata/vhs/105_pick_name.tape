Output "/tmp/ordo-vt/vhs-out/105_pick_name.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

# A unique match opens the project, several open the picker filtered by the name.
Hide
Type "clear && for n in Ferry Ferret Billing; do ./ordo project add $n -q >/dev/null 2>&1; done && ./ordo pick bil"
Enter
Sleep 4000ms
Show
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/105_pick_unique_opens_project.png"

Ctrl+C
Sleep 600ms
Hide
Type "clear && ./ordo pick fer"
Enter
Sleep 2000ms
Show
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/105_pick_several_filter_the_picker.png"

Escape
Sleep 300ms
Ctrl+C
Sleep 600ms
Hide
Type "clear && ./ordo pick zzz"
Enter
Sleep 800ms
Show
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/105_pick_no_match.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms
