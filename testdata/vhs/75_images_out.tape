# Images leaving ordo. ordo task attach stores a file and a piped screenshot,
# and task show lists them by absolute path. A visual y over the crash report
# and its subtask then copies bullets with an indented ![name](/abs/path) line
# per image, and every path it names exists.
Output "/tmp/ordo-vt/vhs-out/75_images_out.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clip.md"

Hide
Type "./ordo category add v0.12.0 -p Default -q"
Enter
Type `./ordo task add "Crash when saving a folded category" -C v0.12.0 -p Default -q --description 'Happens after zc then w.'`
Enter
Type `./ordo task add "Check it on the light theme" --parent "Crash when saving a folded category" -p Default -q`
Enter
Type `./ordo task add "Document the fold keys" -C v0.12.0 -p Default -q`
Enter
Type "./ordo task attach 'Crash when saving a folded category' assets/social-preview.png --name 'Crash dialog' -p Default -q"
Enter
Type "./ordo task attach 'Check it on the light theme' - -p Default -q < assets/demo-light.gif"
Enter
Type "clear && ./ordo task show 'Crash when saving a folded category' -p Default"
Enter
Sleep 800ms
Show
Screenshot "/tmp/ordo-vt/vhs-out/75_show.png"
Sleep 300ms

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Type "/saving a folded"
Enter
Sleep 300ms
Show
Type "v"
Sleep 200ms
Type "jj"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/75_range.png"
Type "y"
Sleep 800ms

Hide
Type "q"
Sleep 300ms
Type `clear && cat /tmp/ordo-vt/clip.md && echo && grep -o "](/[^)]*" /tmp/ordo-vt/clip.md | cut -c3- | xargs -r ls -l`
Enter
Sleep 800ms
Show
Screenshot "/tmp/ordo-vt/vhs-out/75_copied.png"
Sleep 500ms
