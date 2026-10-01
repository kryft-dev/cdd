# Records demo/demo.gif against the environment built by demo/setup.sh.
# setup.sh puts stand-in projects in ~/Code and ~/Work and cdd's own state
# in ~/.cache/cdd-demo, so it never touches your real config or History.
#
#   demo/setup.sh && vhs demo/demo.tape; demo/teardown.sh
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
Set TypingSpeed 90ms

# Hidden setup. cdd's state points at the demo
# directory (Env values are literal, so the shell does it), the demo bin
# comes first on PATH so cdd is the one built from this checkout and nothing
# opens for real, and the prompt is neutral: no username or hostname.
Hide
Set TypingSpeed 5ms
Type "set -gx XDG_CONFIG_HOME $HOME/.cache/cdd-demo/config" Enter
Type "set -gx XDG_DATA_HOME $HOME/.cache/cdd-demo/data" Enter
Type "set -gx XDG_STATE_HOME $HOME/.cache/cdd-demo/state" Enter
Type "set -gx PATH $HOME/.cache/cdd-demo/bin $PATH" Enter
Type "set -g fish_greeting" Enter
Type "set -g fish_autosuggestion_enabled 0" Enter
Type "set -g fish_prompt_pwd_dir_length 0" Enter
Type "function fish_prompt; echo -n (prompt_pwd)' ❯ '; end" Enter
Type "function pwd; builtin pwd | string replace -- $HOME '~'; end" Enter
Type "cdd init fish | source" Enter
Type "cd ~/Code/go" Enter
Type "clear" Enter
Set TypingSpeed 90ms
Show

Sleep 1s
Type "# find every git repo under these dirs" Enter
Sleep 1s
Type "cdd scan ~/Code ~/Work" Sleep 500ms Enter
Sleep 2s

Type "# open the Picker: note the key hints" Enter
Sleep 1s
Type "cdd" Sleep 500ms Enter
Sleep 3.5s
Type "oss kube"
Sleep 3s
Enter
Sleep 500ms
Type "# Enter jumped: where am I?" Enter
Sleep 500ms
Type "pwd" Sleep 500ms Enter
Sleep 2s

Type "# my own Action: ctrl+l runs lazygit" Enter
Sleep 1s
Type "cdd" Sleep 500ms Enter
Sleep 1.5s
Ctrl+l
Sleep 3.5s

Type "# ctrl+y copies the path, ctrl+g opens the remote" Enter
Sleep 1s
Type "cdd" Sleep 500ms Enter
Sleep 1.5s
Ctrl+y
Sleep 2.5s
Ctrl+g
Sleep 1.5s
Escape
Sleep 500ms
Type "# (the opener is a stub: nothing really opened)" Enter
Sleep 500ms
Type "cat ~/.cache/cdd-demo/opened.log" Sleep 500ms Enter
Sleep 2.5s

Type "# add a directory that is not a git repo" Enter
Sleep 1s
Type "cdd add ~/Work/notes" Sleep 500ms Enter
Sleep 1.5s
Type "cdd" Sleep 500ms Enter
Sleep 1.5s
Type "notes"
Sleep 2.5s
Escape
Sleep 500ms

Type "# ctrl+d forgets a Project after y/n" Enter
Sleep 1s
Type "cdd" Sleep 500ms Enter
Sleep 1.5s
Type "rust"
Sleep 1s
Ctrl+d
Sleep 2.5s
Type "y"
Sleep 2.5s
Escape
Sleep 500ms

# Hidden: config init refuses to replace the config the demo used so far.
Hide
Type "rm $XDG_CONFIG_HOME/cdd/config.toml" Enter
Type "clear" Enter
Show
Type "# write a commented example config" Enter
Sleep 1s
Type "cdd config init" Sleep 500ms Enter
Sleep 1.5s
Type "head -20 ~/.cache/cdd-demo/config/cdd/config.toml" Sleep 500ms Enter
Sleep 5s
