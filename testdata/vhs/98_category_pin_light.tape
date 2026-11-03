Output "/tmp/ordo-vt/vhs-out/98_category_pin_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Type "G"
Type "{"
Type "gp"
Type "}"
Type "}"
Type "gp"
Sleep 400ms
Show

# The pin mark reads on a light theme, on the cursor's row and off it.
Screenshot "/tmp/ordo-vt/vhs-out/98_pinned_light.png"
Sleep 300ms
Type "k"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/98_pinned_light_off.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
