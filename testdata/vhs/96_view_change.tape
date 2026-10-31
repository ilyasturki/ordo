Output "/tmp/ordo-vt/vhs-out/96_view_change.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
# Written by rename, as Syncthing does: Fix folded at about 2s, then a filter at about 5s.
Type `clear && (v=/tmp/ordo-vt/data/projects/872e2818-9fd7-44c6-e33e-e6ef4c934ef0.view.json && sleep 2 && echo '{"folds":["a40673ab-453d-45b4-af2c-4e2dc88ae4f9"]}' > $v.new && mv $v.new $v && sleep 3 && echo '{"folds":["a40673ab-453d-45b4-af2c-4e2dc88ae4f9"],"filter":{"statuses":["in_progress"]}}' > $v.new && mv $v.new $v) >/dev/null 2>&1 & ./ordo -p Default`
Enter
Sleep 800ms
Show

Type "jj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/96_before.png"

# No key pressed: the fold applies, the cursor stays on its row, the status line says so.
Sleep 2200ms
Screenshot "/tmp/ordo-vt/vhs-out/96_fold_arrived.png"

# Then the filter.
Sleep 3000ms
Screenshot "/tmp/ordo-vt/vhs-out/96_filter_arrived.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
