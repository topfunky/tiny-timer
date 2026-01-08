# Configuration

Output vhs/basic.gif

Require tiny-timer

Set Shell "bash"
Set FontSize 32
Set Width 1200
Set Height 600

Env PATH "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/vhs"

# Script for recording

Type 'tiny-timer 1 -title "Writing code"' Sleep 1s
Enter Sleep 5s

Type "d" Sleep 1s
Type " and tests" Sleep 3s
Enter Sleep 5s

Type "h" Sleep 3s
Enter Sleep 2s

Type "q"
Sleep 4s
