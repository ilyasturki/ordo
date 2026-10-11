# README demo body, sourced by demo-dark.tape and demo-light.tape.
Set Shell bash
Set FontFamily "IosevkaTerm Nerd Font"
Set FontSize 22
Set LineHeight 1.2
Set Width 1400
Set Height 1000
Set Padding 28
Set Margin 24
Set WindowBar Colorful
Set WindowBarSize 40
Set BorderRadius 10
Set Framerate 25
Set TypingSpeed 40ms
Set CursorBlink false

Hide
Type "source assets/demo/setup.sh && clear && ordo -p ferry"
Enter
Wait+Screen@60s /shipped/
Show
Sleep 2s

# Capture an idea into the next inbox.
Type "j"
Sleep 200ms
Type "a"
Sleep 300ms
Type "Retry uploads on a flaky connection"
Sleep 300ms
Enter
Wait+Screen /flaky connection/
Sleep 700ms

# Send it to the version that ships next.
Type "M"
Wait+Screen /Move to/
Sleep 1s
Enter
Sleep 900ms

# Paste a markdown checklist into the version: the chooser asks how.
Type "}"
Sleep 200ms
Type "}"
Sleep 200ms
Type "k"
Sleep 400ms
Type "p"
Wait+Screen /one task per line/
Sleep 1.5s
Enter
Sleep 1.4s

# Mark part of it done in one go.
Type "k"
Sleep 150ms
Type "v"
Sleep 150ms
Type "k"
Sleep 150ms
Type "k"
Sleep 150ms
Type "k"
Sleep 400ms
Space
Sleep 300ms
Space
Sleep 1s
Type "v"
Sleep 300ms

# Read a long description in place.
Type "/overrides"
Sleep 200ms
Enter
Sleep 400ms
Type "zz"
Sleep 300ms
Type "ze"
Sleep 2.2s

# Delete by mistake, then undo.
Type "/symlink"
Sleep 200ms
Enter
Sleep 500ms
Type "d"
Sleep 700ms
Type "y"
Sleep 800ms
Type "u"
Sleep 1.1s

# Show only what's left.
Type "f"
Sleep 600ms
Enter
Sleep 500ms
Space
Sleep 200ms
Type "j"
Sleep 200ms
Space
Sleep 500ms
Escape
Sleep 500ms
Escape
Sleep 1.4s

# Switch project.
Ctrl+P
Sleep 900ms
Type "/"
Sleep 200ms
Type "home"
Sleep 500ms
Enter
Sleep 2s

Hide
Type "q"
Sleep 500ms
Type 'rm -rf "$demo_dir"'
Enter
