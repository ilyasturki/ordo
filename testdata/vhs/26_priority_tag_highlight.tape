Output "/tmp/phas-vt/vhs-out/26_priority_tag_highlight.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

# Cursor starts on the first task "Build the main dashboard" (high priority ▲).
# On a highlighted row the priority icon must keep its color as a reverse
# color block instead of vanishing into the monochrome selection bar.
Screenshot "/tmp/phas-vt/vhs-out/26_priority_highlight.png"

# Color-only tag (green): the tag block and the priority block sit side by
# side, each in its own color.
Type "t"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/26_coloronly_tag_priority.png"

# The same tag on a task without a priority icon: the block hugs the title.
Type "j"
Sleep 200ms
Type "t"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/26_tag_no_priority.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
