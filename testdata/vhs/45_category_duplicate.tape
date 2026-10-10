Output "/tmp/phas-vt/vhs-out/45_category_duplicate.gif"
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

# Add a category named like an existing one: the editor keeps the text, shows
# the error under it, and the confirm dialog offers the merge.
Type "k"
Type "A"
Sleep 200ms
Type "fix"
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_add_dup_offer.png"
Sleep 300ms

# n closes the offer and leaves the editor open with the error.
Type "n"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_add_dup_editing.png"
Sleep 300ms

# Accepting drops the new row and lands on the existing Fix.
Enter
Sleep 200ms
Type "y"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_add_merged.png"
Sleep 300ms

# Renaming Fix (four tasks) to Ergonomy offers to move its tasks.
Enter
Sleep 200ms
Ctrl+U
Type "ergonomy"
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_rename_offer.png"
Sleep 300ms
Type "n"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_rename_editing.png"
Sleep 300ms

# The tasks land at the end of Ergonomy under a "Fix" separator.
Enter
Sleep 200ms
Enter
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_rename_merged.png"
Sleep 300ms

# One u restores Fix and its tasks.
Type "u"
Sleep 400ms
Screenshot "/tmp/phas-vt/vhs-out/45_rename_undone.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
