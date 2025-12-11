# Output file
Output transaction_debug.gif

# Terminal Styling
Set Shell zsh
Set FontSize 18
Set Width 1200
Set Height 1200
Set Padding 20
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful

# Setup: Hide the setup commands to keep the demo clean
Hide
Type "export PATH=$PATH:$(pwd)/build"
Enter
Type "alias logviewer='./build/logviewer'"
Enter
Type "clear"
Enter
Show

# 1. Search for payment errors in Splunk (The "trigger")
Type "export LOGVIEWER_CONFIG=integration/config.yaml"
Enter
Sleep 1s
Type "logviewer context list"
Sleep 1ms
Enter
Sleep 1s
Type "logviewer query log -i payment-service -f level=ERROR --last 10m --size 5"
Enter
Sleep 4s


Type "logviewer query log -i payment-processor-all -i splunk-all -i opensearch-all -f trace_id=3a0b72f6-af50-4082-b936-9b28aad8df1e --last 10m 
Sleep 1000ms
Enter

Sleep 10s
