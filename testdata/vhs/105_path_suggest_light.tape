Output "/tmp/ordo-vt/vhs-out/105_path_suggest_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Set Theme "Builtin Light"
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "rm -rf /tmp/ordo-vt/tree && mkdir -p /tmp/ordo-vt/tree/notes /tmp/ordo-vt/tree/nixos && touch /tmp/ordo-vt/tree/news.txt"
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# The list of 105_path_suggest on a light theme.
Type "a"
Sleep 200ms
Type "read /tmp/ordo-vt/tree/n"
Sleep 300ms
Down
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/105_light.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Escape
Type "q"
