Output "/tmp/ordo-vt/vhs-out/39_task_command.gif"
Set Shell "bash"
Set Width 1200
Set Height 700
Set FontSize 16
Env ORDO_DATA_PATH "/tmp/ordo-vt/data"
Env ORDO_CONFIG_PATH "/tmp/ordo-vt/cfg"

# Two harmless commands: gn echoes the prompt's answer back, gf always fails.
Hide
Type `printf '%s' '{"show_shortcut_bar": true, "task_commands": [{"key": "gn", "label": "echo", "prompt": "when", "run": ["printenv", "ORDO_INPUT"]}, {"key": "gf", "label": "fail", "run": ["false"]}]}' > /tmp/ordo-vt/cfg/config.json`
Enter
Type "clear && ./ordo -p Default"
Enter
Sleep 800ms
Show

Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/39_task_command_bar.png"

Type "gn"
Sleep 400ms
Type "next friday 14:30"
Sleep 400ms
Screenshot "/tmp/ordo-vt/vhs-out/39_task_command_prompt.png"

Enter
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/39_task_command_result.png"

Type "gf"
Sleep 600ms
Screenshot "/tmp/ordo-vt/vhs-out/39_task_command_failure.png"

Type "?"
Sleep 300ms
Type "/"
Sleep 200ms
Type "echo"
Sleep 500ms
Screenshot "/tmp/ordo-vt/vhs-out/39_task_command_help.png"
# Let the final capture land before quitting (see run.sh).
Sleep 500ms

Hide
Escape
Escape
Type "q"
