package action_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

// fakeBin returns a directory holding an executable for each of names, to
// stand alone as PATH.
func fakeBin(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestClipboardTool_PicksTheFirstFoundInOrder(t *testing.T) {
	all := []string{"wl-copy", "xclip", "xsel", "pbcopy"}
	tests := []struct {
		name    string
		have    []string
		wayland string
		want    []string
	}{
		{"wl-copy on wayland", all, "wayland-0", []string{"wl-copy"}},
		{"wl-copy ignored without wayland", all, "", []string{"xclip", "-selection", "clipboard"}},
		{"xclip when wl-copy is missing", all[1:], "wayland-0", []string{"xclip", "-selection", "clipboard"}},
		{"xsel when xclip is missing", all[2:], "", []string{"xsel", "--clipboard", "--input"}},
		{"pbcopy last", all[3:], "", []string{"pbcopy"}},
		{"none found", nil, "wayland-0", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", fakeBin(t, tt.have...))
			t.Setenv("WAYLAND_DISPLAY", tt.wayland)
			if got := action.ClipboardTool(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClipboardTool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopy_PipesTheTextToTheTool(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "got")
	script := "#!/bin/sh\nread -r line\necho \"$line\" > " + out + "\n"
	if err := os.WriteFile(filepath.Join(dir, "xclip"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "")

	ok, err := action.Copy("/home/me/my proj")
	if !ok || err != nil {
		t.Fatalf("Copy = %v, %v, want true, nil", ok, err)
	}
	got, _ := os.ReadFile(out)
	if strings.TrimSpace(string(got)) != "/home/me/my proj" {
		t.Errorf("tool read %q", got)
	}
}

func TestCopy_NoToolIsNotCopied(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	ok, err := action.Copy("/p")
	if ok || err != nil {
		t.Errorf("Copy = %v, %v, want false, nil", ok, err)
	}
}

func TestCopy_ToolFailureIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "")
	if ok, err := action.Copy("/p"); !ok || err == nil {
		t.Errorf("Copy = %v, %v, want true and an error", ok, err)
	}
}
