Output demo.gif

Set FontSize 16
Set Width 800
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "cat .env"
Enter
Sleep 2s

Type "dotlock init"
Enter
Sleep 1s

Type "mysecretpass"
Enter
Sleep 500ms

Type "mysecretpass"
Enter
Sleep 3s

Type "rm .env .env.prod && echo 'deleted all env files'"
Enter
Sleep 1.5s

Type "dotlock unlock"
Enter
Sleep 2s

Type "cat .env"
Enter
Sleep 3s
