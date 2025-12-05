# Emplacement de sortie
Output demo.gif

Set Shell zsh
Set FontSize 22
Set Width 1600
Set Height 900
Set Padding 40
Set Theme "Catppuccin Mocha"


# --- 1. Window & Layout Aesthetics ---
# Démarrage de la démo
Hide
Type "export PATH=$PATH:$(pwd)/build"
Enter
Type "alias logviewer='./build/logviewer'"
Enter
Type "clear"
Enter
Show

Type "logviewer -c config.yaml -i local-test query log --size 10"
Sleep 500ms
Enter
Sleep 4s
Type "clear"
Enter

Type "logviewer -c config.yaml -i local-test query field"
Sleep 500ms
Enter
Sleep 4s
Type "clear"
Enter

Type "logviewer -c config.yaml -i local-test --inherits pretty-print query log -f level=ERROR"
Sleep 500ms
Enter
Sleep 4s
Type "clear"
Enter