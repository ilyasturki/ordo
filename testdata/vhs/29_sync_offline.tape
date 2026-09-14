Output "/tmp/phas-vt/vhs-out/29_sync_offline.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"

# Enrolled with a server nobody runs: port 1 refuses instantly.
Hide
Type `mkdir -p "$PHASIONARY_STATE_PATH" && printf '{"device_id":"vhs","server_url":"http://127.0.0.1:1","token":"t"}' > "$PHASIONARY_STATE_PATH/device.json"`
Enter
Type "clear && ./phasionary -p Default"
Enter
Sleep 1200ms
Show

Screenshot "/tmp/phas-vt/vhs-out/29_sync_offline_toast.png"
Sleep 300ms

Type "j"
Sleep 300ms
Screenshot "/tmp/phas-vt/vhs-out/29_sync_offline_tag.png"
Sleep 300ms

Hide
Type "q"
Sleep 500ms
Type `rm -f "$PHASIONARY_STATE_PATH/device.json"`
Enter
