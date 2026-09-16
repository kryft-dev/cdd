package history

import (
	"os"
	"path/filepath"
)

// DefaultPath returns the default History file path:
// $XDG_DATA_HOME/cdd/history, falling back to ~/.local/share/cdd/history
// when XDG_DATA_HOME is unset.
func DefaultPath() (string, error) {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "cdd", "history"), nil
}
