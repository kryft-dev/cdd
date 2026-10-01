package action_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
)

func builtin(t *testing.T, name string) action.Action {
	t.Helper()
	for _, a := range action.Builtins() {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("no built-in Action %q", name)
	return action.Action{}
}

func ptr[T any](v T) *T { return &v }

func TestBuiltins_FilesOpensTheProjectDetached(t *testing.T) {
	got := builtin(t, "files")
	want := action.Action{Name: "files", Key: "ctrl+o", Run: action.Opener(runtime.GOOS) + " {path}", Detach: true}
	if got != want {
		t.Errorf("files = %+v, want %+v", got, want)
	}
}

func TestBuiltins_EditorIsBoundAndNotDetached(t *testing.T) {
	got := builtin(t, "editor")
	if got.Key != "ctrl+e" || got.Detach || got.Jump || got.Run == "" {
		t.Errorf("editor = %+v, want ctrl+e, a command, attached", got)
	}
}

// The editor is looked up by the shell when the Action runs, so these run
// the real command with a stand-in for each editor.
func TestBuiltins_EditorPrefersVisualThenEditorThenVi(t *testing.T) {
	tests := []struct {
		name           string
		visual, editor string
		want           string
	}{
		{"visual wins", "echo visual", "echo editor", "visual"},
		{"editor when visual is empty", "", "echo editor", "editor"},
		{"vi when neither is set", "", "", "vi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)
			bin := t.TempDir()
			if err := os.WriteFile(bin+"/vi", []byte("#!/bin/sh\necho vi\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
			dir, tty := spacedDir(t), fakeTTY(t)

			code, err := action.ExecRunner{TTY: tty}.Run(builtin(t, "editor"), dir)
			if err != nil || code != 0 {
				t.Fatalf("Run = %d, %v", code, err)
			}
			out, _ := os.ReadFile(tty)
			if got := strings.TrimSpace(string(out)); !strings.HasPrefix(got, tt.want) {
				t.Errorf("output = %q, want it to start with %q", got, tt.want)
			}
		})
	}
}

func TestBuiltins_EditorGetsTheQuotedPathAfterItsOwnWords(t *testing.T) {
	t.Setenv("VISUAL", "printf %s, ")
	dir, tty := spacedDir(t), fakeTTY(t)

	if _, err := (action.ExecRunner{TTY: tty}).Run(builtin(t, "editor"), dir); err != nil {
		t.Fatal(err)
	}
	if out, _ := os.ReadFile(tty); string(out) != dir+"," {
		t.Errorf("output = %q, want the path as one last argument", out)
	}
}

func TestMerge_UserOverridesFilesAndEditor(t *testing.T) {
	got, err := action.Merge(map[string]action.Override{
		"files":  {Run: ptr("thunar {path}"), Key: ptr("ctrl+f")},
		"editor": {Run: ptr("hx {path}")},
	}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	byName := map[string]action.Action{}
	for _, a := range got {
		byName[a.Name] = a
	}
	if f := byName["files"]; f.Run != "thunar {path}" || f.Key != "ctrl+f" || !f.Detach {
		t.Errorf("files = %+v", f)
	}
	if e := byName["editor"]; e.Run != "hx {path}" || e.Key != "ctrl+e" || e.Detach {
		t.Errorf("editor = %+v", e)
	}
}

func TestMerge_FilesAndEditorCanBeUnbound(t *testing.T) {
	got, err := action.Merge(map[string]action.Override{
		"files":  {Key: ptr("")},
		"editor": {Key: ptr("")},
	}, false)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	for _, a := range got {
		if (a.Name == "files" || a.Name == "editor") && a.Key != "" {
			t.Errorf("%s key = %q, want unbound", a.Name, a.Key)
		}
	}
}
