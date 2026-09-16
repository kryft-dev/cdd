package history_test

import (
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/history"
)

func TestDefaultPath_UsesXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")

	got, err := history.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: unexpected error: %v", err)
	}
	want := filepath.Join("/xdg/data", "cdd", "history")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPath_FallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/tester")

	got, err := history.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: unexpected error: %v", err)
	}
	want := filepath.Join("/home/tester", ".local", "share", "cdd", "history")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}
