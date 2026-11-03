Output "/tmp/ordo-vt/vhs-out/45_category_duplicate.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"
Env ORDO_CLIPBOARD_FILE "/tmp/ordo-vt/clipboard"

Hide
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

# Add a category named like an existing one: the editor keeps the text, shows
# the error under it, and the chooser offers a free name, a rename of the
# other one, or the merge.
Type "k"
Type "A"
Sleep 200ms
Type "fix"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_dup_offer.png"
Sleep 300ms

# The merge choice says what it does.
Type "jj"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_dup_merge_choice.png"
Sleep 300ms

# esc closes the chooser and leaves the editor open with the error.
Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_dup_editing.png"
Sleep 300ms

# t names the new one "fix (2)".
Enter
Sleep 200ms
Type "t"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_named_two.png"
Sleep 300ms

# Another add of "fix", then o: the existing Fix becomes "Fix (3)".
Type "A"
Sleep 200ms
Type "fix"
Enter
Sleep 200ms
Type "o"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_renamed_other.png"
Sleep 300ms
Type "u"
Type "u"
Sleep 300ms

# Accepting the merge drops the new row and lands on the existing Fix.
Type "gg"
Type "A"
Sleep 200ms
Type "fix"
Enter
Sleep 200ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_add_merged.png"
Sleep 300ms

# Renaming Fix (four tasks) to Ergonomy, then m: the tasks land at the end of
# Ergonomy under a "Fix" separator.
Enter
Sleep 200ms
Ctrl+U
Type "ergonomy"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_rename_offer.png"
Sleep 300ms
Type "m"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_rename_merged.png"
Sleep 300ms

# One u restores Fix and its tasks.
Type "u"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/45_rename_undone.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
