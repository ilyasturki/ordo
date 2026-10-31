Output "/tmp/ordo-vt/vhs-out/94_outside_change.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
# sed -i replaces the file by rename, as Syncthing does: at about 2s, then 6s.
Type `clear && (sleep 2 && f=$(ls /tmp/ordo-vt/data/projects/*.json) && sed -i 's/Build the main dashboard/Changed by another device/' "$f" && sleep 4 && sed -i 's/Polish help overlay styling/Changed behind the dialog/' "$f") >/dev/null 2>&1 & ./ordo -p Default`
Enter
Sleep 800ms
Show

Type "jj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/94_before.png"

# No key pressed: the row changes, the cursor stays on its row, the status line says so.
Sleep 2500ms
Screenshot "/tmp/ordo-vt/vhs-out/94_reloaded.png"

# A dialog open when the file changes stays as it is.
Type "?"
Sleep 3500ms
Screenshot "/tmp/ordo-vt/vhs-out/94_dialog_undisturbed.png"

# Closing it applies the change.
Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/94_dialog_closed.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
