package action_test

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/git"
)

// repoWithRemote makes a git repository whose origin is url, or that has no
// remote when url is empty, and returns its path.
func repoWithRemote(t *testing.T, url string) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{{"init", "-q"}}
	if url != "" {
		cmds = append(cmds, []string{"remote", "add", "origin", url})
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestBuiltins_RemoteOpensTheRemoteDetached(t *testing.T) {
	got := builtin(t, "remote")
	want := action.Action{Name: "remote", Key: "ctrl+g", Run: action.Opener(runtime.GOOS) + " {remote}", Detach: true}
	if got != want {
		t.Errorf("remote = %+v, want %+v", got, want)
	}
}

func TestExecRunner_RemoteExpandsToTheQuotedHomePage(t *testing.T) {
	dir := repoWithRemote(t, "git@github.com:owner/repo.git")
	tty := fakeTTY(t)

	a := action.Action{Run: `printf '%s' {remote}`}
	if _, err := (action.ExecRunner{TTY: tty}).Run(a, dir); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(tty)
	if want := "https://github.com/owner/repo"; string(out) != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestExecRunner_StartFailsWithoutARemote(t *testing.T) {
	notRepo := t.TempDir()
	noRemote := repoWithRemote(t, "")

	a := action.Action{Run: "true {remote}"}
	if err := (action.ExecRunner{}).Start(a, notRepo); !errors.Is(err, git.ErrNotRepo) {
		t.Errorf("Start outside a repo: err = %v, want ErrNotRepo", err)
	}
	if err := (action.ExecRunner{}).Start(a, noRemote); !errors.Is(err, git.ErrNoRemote) {
		t.Errorf("Start without a remote: err = %v, want ErrNoRemote", err)
	}
}

func TestExecRunner_ARunWithoutRemoteNeverLooksItUp(t *testing.T) {
	a := action.Action{Run: "true"}
	if err := (action.ExecRunner{}).Start(a, t.TempDir()); err != nil {
		t.Errorf("Start = %v, want nil: no {remote} to resolve", err)
	}
}
