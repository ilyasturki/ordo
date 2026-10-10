Output "/tmp/phas-vt/vhs-out/41_picker_ctrl_c.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# ctrl+c in the add input of `pick` quits instead of being typed.
Hide
Type "clear && ./phasionary pick"
Enter
Sleep 1500ms
Show
Type "a"
Sleep 300ms
Type "Half typed"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/41_pick_add_input.png"

Ctrl+C
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/41_pick_add_ctrl_c_quits.png"
Sleep 500ms

# The full-screen picker opened from a project quits on ctrl+c too.
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Ctrl+P
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/41_picker_fullscreen.png"

Ctrl+C
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/41_picker_ctrl_c_quits.png"
Sleep 500ms

# In the description editor ctrl+c saves the text, then quits.
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Type "zd"
Sleep 400ms
Type "j"
Sleep 300ms
Enter
Sleep 500ms
Type " kept by ctrl+c"
Sleep 300ms
Ctrl+C
Sleep 600ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Type "zd"
Sleep 400ms
Type "j"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/41_description_ctrl_c_saved.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
