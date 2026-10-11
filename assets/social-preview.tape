# GitHub social preview (Settings → Social preview), 1280×640. Run `just demo`.
Set Shell bash
Set FontFamily "IosevkaTerm Nerd Font"
Set FontSize 22
Set LineHeight 1.2
Set Width 1280
Set Height 640
Set Padding 28
Set Margin 24
Set MarginFill "#0d1117"
Set WindowBar Colorful
Set WindowBarSize 40
Set BorderRadius 10
Set Theme { "name": "phasionary-dark", "background": "#1a1b26", "foreground": "#c0caf5", "cursor": "#c0caf5", "selection": "#33467c", "black": "#15161e", "red": "#f7768e", "green": "#9ece6a", "yellow": "#e0af68", "blue": "#7aa2f7", "magenta": "#bb9af7", "cyan": "#73daca", "white": "#a9b1d6", "brightBlack": "#565f89", "brightRed": "#ff899d", "brightGreen": "#afd67a", "brightYellow": "#ffc777", "brightBlue": "#8db0ff", "brightMagenta": "#c7a9ff", "brightCyan": "#7fe4d2", "brightWhite": "#c0caf5" }

Hide
Type "source assets/demo/setup.sh && clear && phasionary -p ferry"
Enter
Wait+Screen@60s /v1\.3\.0/
Type "}ztj"
Sleep 500ms
Show
Sleep 300ms
Screenshot assets/social-preview.png
Sleep 500ms

Hide
Type "q"
Sleep 500ms
Type 'rm -rf "$demo_dir"'
Enter
