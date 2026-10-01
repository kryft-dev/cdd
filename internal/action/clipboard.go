package action

import (
	"os"
	"os/exec"
	"strings"
)

// ClipboardTool returns the command line of the first clipboard program
// found on PATH, in this order: wl-copy (only when WAYLAND_DISPLAY is set),
// xclip, xsel, pbcopy. Each reads the text from stdin. It returns nil when
// none is found, which is when the caller falls back to the OSC 52 escape.
func ClipboardTool() []string {
	tools := [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"pbcopy"},
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		tools = append([][]string{{"wl-copy"}}, tools...)
	}
	for _, t := range tools {
		if _, err := exec.LookPath(t[0]); err == nil {
			return t
		}
	}
	return nil
}

// Copy puts text on the clipboard with the program ClipboardTool picks. It
// reports false, without an error, when there is none, and true with the
// error when the program was found but failed.
func Copy(text string) (bool, error) {
	tool := ClipboardTool()
	if tool == nil {
		return false, nil
	}
	cmd := exec.Command(tool[0], tool[1:]...)
	cmd.Stdin = strings.NewReader(text)
	return true, cmd.Run()
}
