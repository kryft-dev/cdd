package action_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/action"
)

// spacedDir makes a Project directory whose path has a space and a quote.
func spacedDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "my proj's dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// fakeTTY creates an empty file for ExecRunner.Run to stand in for
// /dev/tty, and returns its path.
func fakeTTY(t *testing.T) string {
	t.Helper()
	tty := filepath.Join(t.TempDir(), "tty")
	if err := os.WriteFile(tty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return tty
}

func TestExecRunner_RunQuotesPathAndSetsEnvAndCwd(t *testing.T) {
	dir := spacedDir(t)
	tty := fakeTTY(t)
	r := action.ExecRunner{TTY: tty}

	a := action.Action{Run: `printf '%s|%s|%s' {path} "$CDD_PATH" "$PWD"`}
	code, err := r.Run(a, dir)
	if err != nil || code != 0 {
		t.Fatalf("Run = %d, %v", code, err)
	}

	out, _ := os.ReadFile(tty)
	if want := strings.Repeat(dir+"|", 2) + dir; string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestExecRunner_RunReturnsExitStatus(t *testing.T) {
	r := action.ExecRunner{TTY: fakeTTY(t)}

	code, err := r.Run(action.Action{Run: "exit 3"}, t.TempDir())
	if err != nil || code != 3 {
		t.Errorf("Run = %d, %v, want 3, nil", code, err)
	}
}

func TestExecRunner_RunFailsWithoutATerminal(t *testing.T) {
	r := action.ExecRunner{TTY: filepath.Join(t.TempDir(), "missing", "tty")}

	if _, err := r.Run(action.Action{Run: "true"}, t.TempDir()); err == nil {
		t.Error("Run: want an error when the terminal cannot be opened")
	}
}

func TestExecRunner_StartRunsWithoutWaiting(t *testing.T) {
	dir := spacedDir(t)

	a := action.Action{Run: `printf '%s' {path} > "$CDD_PATH/out"; printf noise`, Detach: true}
	if err := (action.ExecRunner{}).Start(a, dir); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var out []byte
	for i := 0; i < 200 && string(out) != dir; i++ {
		time.Sleep(10 * time.Millisecond)
		out, _ = os.ReadFile(filepath.Join(dir, "out"))
	}
	if string(out) != dir {
		t.Errorf("out = %q, want %q", out, dir)
	}
}

func TestExecRunner_StartReportsAFailureToStart(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")

	if err := (action.ExecRunner{}).Start(action.Action{Run: "true"}, missing); err == nil {
		t.Error("Start: want an error for a Project directory that does not exist")
	}
}
