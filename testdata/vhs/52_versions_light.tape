Output "/tmp/phas-vt/vhs-out/52_versions_light.gif"
Set Shell "bash"
Set Width 1200
Set Height 900
Set FontSize 16
Set Theme "Builtin Light"
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "./phasionary import testdata/vhs/releases.json -q && clear && ./phasionary -p Releases"
Enter
Sleep 800ms
Show

# The dimmed shipped stack and the warning badge stay legible on a light theme.
Screenshot "/tmp/phas-vt/vhs-out/52_overview_light.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
