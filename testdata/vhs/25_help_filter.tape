Output "/tmp/ordo-vt/vhs-out/25_help_filter.gif"
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

Type "?"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_open.png"

Type "/"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_open.png"

Type "add"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_by_label.png"

Escape
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_cleared.png"

# A query that only matches sections the card doesn't show.
Type "/"
Sleep 200ms
Type "markdown"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_hidden_section.png"

Escape
Sleep 200ms
Type "/"
Sleep 200ms
Type "zzzznope"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_no_match.png"

Escape
Sleep 200ms
Type "/"
Sleep 200ms
Type "opt"
Sleep 400ms
Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/25_help_filter_enter_runs.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Type "q"
