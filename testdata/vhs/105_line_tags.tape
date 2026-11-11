Output "/tmp/ordo-vt/vhs-out/105_line_tags.gif"
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

# A labelled separator below the first task, and t marks it like a task.
Type "j"
Sleep 200ms
Type "-"
Sleep 200ms
Enter
Type "Next up"
Enter
Sleep 300ms
Type "t"
Sleep 300ms

# A note below it, and T picks its mark.
Type "_"
Sleep 300ms
Type "Check the old totals first"
Enter
Sleep 300ms
Type "T"
Sleep 300ms
Type "jj"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/105_note_picker.png"
Sleep 300ms
Enter
Sleep 300ms
Type "j"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_tagged_lines.png"
Sleep 500ms

# The filter hub, then the tag list: the green separator alone matches it.
Type "f"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_filter_hub.png"
Type "jjj"
Enter
Sleep 300ms
Type "j"
Type " "
Sleep 300ms
Type "q"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/105_tag_filter.png"
Sleep 500ms

Hide
Type "q"
Enter
Sleep 300ms
