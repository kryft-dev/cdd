package config_test

import (
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/config"
)

// TestPath checks Path honours XDG_CONFIG_HOME and falls back to
// ~/.config.
func TestPath(t *testing.T) {
	t.Run("xdg", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)

		got, err := config.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(xdg, "cdd", "config.toml"); got != want {
			t.Errorf("Path = %q, want %q", got, want)
		}
	})

	t.Run("home fallback", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", home)

		got, err := config.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if want := filepath.Join(home, ".config", "cdd", "config.toml"); got != want {
			t.Errorf("Path = %q, want %q", got, want)
		}
	})
}
