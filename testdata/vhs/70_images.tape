# Images on tasks. A PNG stands in for the clipboard (copied out of the repo,
# so nothing writes over the asset): p attaches it to the focused task, whose
# title row then shows ▣. zd shows the description's chip, gi lists the image's
# path, size and pixel size, and a task referencing an image that hasn't
# synced yet reads "(not synced yet)", in the list and in gi.
Output "/tmp/ordo-vt/vhs-out/70_images.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clip.png"

Hide
Type "cp assets/social-preview.png /tmp/ordo-vt/clip.png"
Enter
Type `./ordo task add "Fix the overflow on mobile" -C Feature -p Default -q --description 'Seen on the laptop:'$'\n''![Laptop screenshot](attachments/ffffffffffffffffffffffffffffffff.png)'`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

Type "j"
Sleep 200ms
Type "p"
Sleep 2000ms
Screenshot "/tmp/ordo-vt/vhs-out/70_attached.png"

Type "zd"
Sleep 1200ms
Screenshot "/tmp/ordo-vt/vhs-out/70_chips.png"

Type "gi"
Sleep 1200ms
Screenshot "/tmp/ordo-vt/vhs-out/70_info.png"

Type "q"
Sleep 200ms
Type "/overflow"
Enter
Sleep 300ms
Type "gi"
Sleep 1200ms
Screenshot "/tmp/ordo-vt/vhs-out/70_info_missing.png"
Sleep 500ms

Hide
Type "q"
Type "q"
