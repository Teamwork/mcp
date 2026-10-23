# Run from plugins/review after exporting credentials privately in the parent shell.
# This records live agent/tool events. It never inserts canned responses.
Output teamwork-review.mp4
Output teamwork-review.webm
Output "teamwork-review-frames/"
Require python3
Require codex
Set Shell "bash"
Set FontSize 19
Set Width 1600
Set Height 1000
Set Framerate 15
Set Theme "Catppuccin Mocha"
Set Padding 24
Set TypingSpeed 15ms
Set WaitTimeout 90m
Type "python3 demo.py"
Enter
Wait+Screen /REVIEW_RUN_FINISHED|BLOCKED:/
Sleep 5s
