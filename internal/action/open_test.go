package action_test

import (
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

func TestOpener_PicksThePlatformProgram(t *testing.T) {
	tests := []struct{ goos, want string }{
		{"linux", "xdg-open"},
		{"darwin", "open"},
		{"freebsd", "xdg-open"},
	}
	for _, tt := range tests {
		if got := action.Opener(tt.goos); got != tt.want {
			t.Errorf("Opener(%q) = %q, want %q", tt.goos, got, tt.want)
		}
	}
}
