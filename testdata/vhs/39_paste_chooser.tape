Output "/tmp/phas-vt/vhs-out/39_paste_chooser.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# Seeds the SYSTEM clipboard with a note, then lands it each way the chooser
# offers besides one task per line (34_paste_lines). Your clipboard's text is
# saved first and put back at the end.
Hide
Type "clear && timeout 2 wl-paste -n > /tmp/phas-vt/clip.bak 2>/dev/null; printf 'Call the bank\n\n  ask about the fees\n  and the new card\n' | timeout 2 wl-copy"
Enter
Sleep 500ms
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

# Descriptions open up front: zd keeps the cursor's row index, not its task.
Type "zd"
Sleep 300ms
Type "jj"
Sleep 300ms
Type "p"
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/39_chooser.png"

# d: one task, the first line its title and the rest its description.
Type "d"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/39_title_and_description.png"
Type "u"
Sleep 300ms

# o: one task, every line joined into its title.
Type "p"
Sleep 700ms
Type "o"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/39_joined.png"
Type "u"
Sleep 300ms

# a: the note appended to the focused task's description.
Type "p"
Sleep 700ms
Type "a"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/39_appended.png"
Type "u"
Sleep 300ms

# In the title editor ctrl+v reads the same clipboard and asks too; o drops
# the joined lines at the cursor and the edit goes on.
Type "a"
Sleep 300ms
Type "Note: "
Ctrl+V
Sleep 700ms
Screenshot "/tmp/phas-vt/vhs-out/39_editor_chooser.png"
Type "o"
Sleep 500ms
Screenshot "/tmp/phas-vt/vhs-out/39_editor_joined.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Type "q"
Sleep 300ms
Type "clear && ([ -s /tmp/phas-vt/clip.bak ] && timeout 2 wl-copy < /tmp/phas-vt/clip.bak || timeout 2 wl-copy --clear)"
Enter
Sleep 300ms
