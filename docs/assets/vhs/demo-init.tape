# VHS tape: ralph init demo
# Render with: vhs docs/assets/vhs/demo-init.tape
# Output:      docs/assets/demo-init.gif

Output docs/assets/demo-init.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Theme "GitHub Dark"
Set Padding 20
Set TypingSpeed 40ms
Set PlaybackSpeed 1.0

Hide
Type "cd $(mktemp -d)" Enter
Show

Type "ralph init my-project" Sleep 400ms Enter
Sleep 4s

Type "cd my-project" Enter
Sleep 500ms

Type "ls -la" Enter
Sleep 2s

Type "ralph doctor" Enter
Sleep 3s
