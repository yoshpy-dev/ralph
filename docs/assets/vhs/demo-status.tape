# VHS tape: ralph status TUI demo
# Render with: vhs docs/assets/vhs/demo-status.tape
# Output:      docs/assets/demo-status.gif
#
# Prerequisite: run inside a project that has an active Ralph Loop plan
# (e.g., `./scripts/new-ralph-plan.sh sample N/A 3` and at least one iteration).

Output docs/assets/demo-status.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 820
Set Theme "GitHub Dark"
Set Padding 20
Set TypingSpeed 40ms

Type "ralph status" Enter
Sleep 3s

Tab
Sleep 1s
Tab
Sleep 1s
Tab
Sleep 1s
Tab
Sleep 1s

Down Down Down
Sleep 1500ms

Type "q"
Sleep 500ms
