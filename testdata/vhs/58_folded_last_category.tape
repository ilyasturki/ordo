# Stepping onto a folded last category (header + "(folded)") from a scrolled view shows it.
# 24 rows: Height 500 with Padding 10 at FontSize 16 (1 row = 20px, rows = (Height-20)/20).
Output "/tmp/ordo-vt/vhs-out/58_folded_last_category.gif"
Set Shell "bash"
Set Width 1000
Set Height 500
Set FontSize 16
Set Padding 10
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Fold Research (the last category), then walk down onto it.
Type "G"
Tab
Sleep 200ms
Type "gg"
Type@40ms "jjjjjjjjjjjjjjjjjj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/58_folded_last.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
