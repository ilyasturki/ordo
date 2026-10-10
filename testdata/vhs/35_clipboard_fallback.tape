# The real clipboard route with no clipboard utility reachable: no clipboard
# file, WAYLAND_DISPLAY empty and XDG_RUNTIME_DIR pointing at a directory with
# no compositor socket in it, so no wl-paste can start. A p with nothing copied
# says it can't read the clipboard; y still copies, as OSC 52; the next p
# pastes that copy, which an unreadable clipboard can't contradict.
Output "/tmp/phas-vt/vhs-out/35_clipboard_fallback.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env PHASIONARY_DATA_PATH "/tmp/phas-vt/data"
Env PHASIONARY_CONFIG_PATH "/tmp/phas-vt/cfg"
Env PHASIONARY_CLIPBOARD_FILE ""
Env XDG_RUNTIME_DIR "/tmp/phas-vt/no-wayland-socket"
Env WAYLAND_DISPLAY ""

Hide
Type "clear && ./phasionary -p Default"
Enter
Sleep 800ms
Show

Type "j"
Sleep 200ms
Type "p"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/35_unreadable_clipboard.png"

Type "y"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/35_clipboard_fallback.png"

Type "p"
Sleep 600ms
Screenshot "/tmp/phas-vt/vhs-out/35_pasted_the_copy.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
