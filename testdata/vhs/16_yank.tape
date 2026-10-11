Output "/tmp/ordo-vt/vhs-out/16_yank.gif"
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

# Add a task with an embedded UUID and a URL so gy lists several parts.
Type "a"
Sleep 200ms
Type "Migrate record 550e8400-e29b-41d4-a716-446655440000 see https://example.com"
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/16_yank_task_added.png"

# Open the yank picker with gy (cursor lands on the new task).
Type "gy"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/16_yank_picker.png"

# Move down to the detected UUID entry.
Type "j"
Sleep 300ms
Screenshot "/tmp/ordo-vt/vhs-out/16_yank_picker_uuid.png"

# Copy the selected part; status line confirms the copy.
Enter
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/16_yank_copied.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
