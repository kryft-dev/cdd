# Records demo/demo.gif against the environment built by demo/setup.sh.
# Run `demo/setup.sh` first; it never touches your real Root or History.
#
#   demo/setup.sh && vhs demo/demo.tape
#
# Use VHS v0.11.0 (go install github.com/charmbracelet/vhs@v0.11.0).
# v0.12.0 cancels its own context before rendering and silently writes
# no GIF.

Output demo/demo.gif

Set Shell fish
Set FontSize 16
Set Width 1200
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

Env XDG_CONFIG_HOME "/tmp/cdd-demo/xdg/config"
Env XDG_DATA_HOME "/tmp/cdd-demo/xdg/data"

Hide
Type "set -gx PATH /tmp/cdd-demo/bin $PATH" Enter
Type "set -g fish_autosuggestion_enabled 0" Enter
Type "cdd init fish | source" Enter
Type "cd /tmp/cdd-demo/root/lab/notes" Enter
Type "clear" Enter
Show

Type "cdd" Sleep 800ms Enter
Sleep 2.5s
Down Sleep 600ms
Down Sleep 600ms
Type "grg" Sleep 1.5s
Enter
Sleep 800ms
Type "pwd" Enter
Sleep 2.5s
