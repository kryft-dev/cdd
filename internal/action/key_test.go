package action_test

import (
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		key  string
		vim  bool
		want string // "" with wantErr false is an unbound key
		err  bool
	}{
		{key: ""},
		{key: "ctrl+v", want: "ctrl+v"},
		{key: "alt+x", want: "alt+x"},
		{key: "alt+ctrl+x", want: "ctrl+alt+x"},
		{key: "enter", want: "enter"},
		{key: "f1", want: "f1"},
		{key: "f20", want: "f20"},
		{key: "shift+tab", want: "shift+tab"},
		{key: "ctrl++", err: true},
		{key: "alt++", want: "alt++"},
		{key: "ctrl+", err: true},
		{key: "f21", err: true},
		{key: "f", err: true},
		{key: "ctrl+1", err: true},
		{key: "shift+a", err: true},
		{key: "ctrl+ctrl+a", err: true},
		{key: "hyper+a", err: true},
		{key: "nonsense", err: true},
		{key: "esc", vim: true, err: true},
		{key: "ctrl+c", vim: true, err: true},
		{key: "?", err: true},
		{key: "?", vim: true, want: "?"},
		{key: "G", vim: true, want: "G"},
		{key: "space", err: true},
	}
	for _, tt := range tests {
		got, err := action.NormalizeKey(tt.key, tt.vim)
		if (err != nil) != tt.err {
			t.Errorf("NormalizeKey(%q, vim=%v) error = %v, want error %v", tt.key, tt.vim, err, tt.err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeKey(%q, vim=%v) = %q, want %q", tt.key, tt.vim, got, tt.want)
		}
	}
}
