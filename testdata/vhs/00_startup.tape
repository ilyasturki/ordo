Output "/tmp/ordo-vt/vhs-out/00_startup.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

Screenshot "/tmp/ordo-vt/vhs-out/00_startup.png"
Sleep 300ms

Hide
Type "q"
