Output "/tmp/ordo-vt/vhs-out/122_title_split.gif"
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

# A new task's title runs on into what should be its description.
Type "zd"
Sleep 300ms
Type "a"
Sleep 300ms
Type "Fix the ferry timetable parser it breaks on the DST switch and on Sundays"
Sleep 300ms
# Back to where the title should end, before "it".
Left 42
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/122_cursor_placed.png"
Sleep 200ms

# shift+enter (ESC+CR from kitty and ghostty): the rest moves to the description,
# whose editor opens with the cursor after the moved text.
Alt+Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/122_moved.png"
Sleep 200ms

# Writing goes on there, and esc saves it like enter.
Type ". Seen on the 6:40 boat."
Sleep 200ms
Escape
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/122_saved_on_esc.png"
Sleep 200ms

# u takes the typing back, and a second u the add with its split.
Type "u"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/122_typing_undone.png"
Sleep 200ms
Type "u"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/122_add_undone.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
