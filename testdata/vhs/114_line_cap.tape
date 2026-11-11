Output "/tmp/ordo-vt/vhs-out/114_line_cap.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# z+ and z- step every title and description through 3, 6, 10 rows and whole.
# Seed a long title and grow the dashboard description: 9 seeded lines + 12.
Hide
Type `clear && ./ordo task add "$(for i in $(seq 1 120); do printf 'w%03d ' $i; done)END" -C Feature -q`
Enter
Sleep 500ms
Type `clear && sed -i 's/how much history to keep"/how much history to keep'"$(for i in $(seq -w 1 12); do printf '\\\\n- note %s' $i; done)"'"/' /tmp/ordo-vt/data/projects/*.json`
Enter
Sleep 300ms
Type "clear && ./ordo -p Default"
Enter
Sleep 1000ms
Show

Type "zd"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/114_cap_3.png"

Type "z+"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/114_cap_6.png"

Type "z+"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/114_cap_10.png"

Type "z+"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/114_cap_whole.png"

Type "z-z-z-"
Sleep 800ms
Screenshot "/tmp/ordo-vt/vhs-out/114_cap_back_to_3.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
