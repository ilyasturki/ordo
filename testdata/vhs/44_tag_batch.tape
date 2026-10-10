Output "/tmp/phas-vt/vhs-out/44_tag_batch.gif"
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

# Visual t on the first three tasks: the block shares one fresh mark (green).
Type "v"
Type "jj"
Type "t"
Sleep 200ms
Escape
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/44_first_batch.png"
Sleep 300ms

# t on an untagged task skips green, which open tasks in Feature use.
Type "j"
Type "t"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/44_next_free.png"
Sleep 300ms

# T on a green task offers the whole batch; space ticks it.
Type "kkk"
Type "T"
Sleep 200ms
Type " "
Sleep 200ms
Type "jjjjjjj"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/44_batch_picker.png"
Sleep 300ms

# Enter recolours all three green tasks to the cyan diamond.
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/44_batch_recoloured.png"
Sleep 300ms

# One u brings the whole batch back to green.
Type "u"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/44_batch_undone.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
