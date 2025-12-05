# Emplacement de sortie
Output demo.gif

Set Shell zsh
Set FontSize 20


# --- 1. Window & Layout Aesthetics ---
# Démarrage de la démo
Hide
Type "export COLUMNS=160 LINES=40"
Enter
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

Type "logviewer -c config.yaml -i local-test query log -f Level=ERROR"
Sleep 500ms
Enter
Sleep 4s
Type "clear"
Enter

Type 'logviewer -c config.yaml -i local-test query log -f Level=INFO --format "[{{.Timestamp.Format \"15:04:05\"}}] {{.Fields.Thread}} -> {{.Message}}"'
Sleep 500ms
Enter
Sleep 5s