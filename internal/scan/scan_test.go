package scan_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/scan"
)

// runGit runs a git command in dir with a test identity and no system or
// global config, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME=/nonexistent-cdd-test-home",
		"GIT_AUTHOR_NAME=cdd test",
		"GIT_AUTHOR_EMAIL=cdd-test@example.com",
		"GIT_COMMITTER_NAME=cdd test",
		"GIT_COMMITTER_EMAIL=cdd-test@example.com",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// newProjectDir creates a directory for a Project at kind/name under root.
func newProjectDir(t *testing.T, root, kind, name string) string {
	t.Helper()

	dir := filepath.Join(root, kind, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return dir
}

// newRepoProjectDir creates a Project directory that is also a git
// repository with one commit, and returns its path.
func newRepoProjectDir(t *testing.T, root, kind, name string) string {
	t.Helper()

	dir := newProjectDir(t, root, kind, name)
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.name", "cdd test")
	runGit(t, dir, "config", "user.email", "cdd-test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// newHistory opens an empty History in a fresh temp directory.
func newHistory(t *testing.T) *history.History {
	t.Helper()

	path := filepath.Join(t.TempDir(), "history")
	hist, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("history.Open: %v", err)
	}
	return hist
}

func newConfig(root string, exclude []string) config.Config {
	return config.Config{Root: root, Exclude: exclude}
}

func TestRun_RepositorySeededFromCommitTime(t *testing.T) {
	root := t.TempDir()
	dir := newRepoProjectDir(t, root, "tools", "cdd")

	commitTime, ok, err := git.LastCommit(context.Background(), dir)
	if err != nil || !ok {
		t.Fatalf("git.LastCommit: ok=%v err=%v", ok, err)
	}

	hist := newHistory(t)
	summary, err := scan.Run(context.Background(), newConfig(root, nil), hist)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary != (scan.Summary{Seeded: 1, Projects: 1}) {
		t.Fatalf("summary = %+v, want {Seeded:1 Projects:1}", summary)
	}

	visits, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(visits) != 1 {
		t.Fatalf("len(visits) = %d, want 1", len(visits))
	}
	if visits[0].Project != "tools/cdd" {
		t.Fatalf("project = %q, want tools/cdd", visits[0].Project)
	}
	if visits[0].Source != history.SourceScan {
		t.Fatalf("source = %q, want scan", visits[0].Source)
	}
	if !visits[0].At.Equal(commitTime) {
		t.Fatalf("At = %v, want %v", visits[0].At, commitTime)
	}
}

func TestRun_NonRepositorySeededFromMtime(t *testing.T) {
	root := t.TempDir()
	dir := newProjectDir(t, root, "tools", "scratch")

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	hist := newHistory(t)
	summary, err := scan.Run(context.Background(), newConfig(root, nil), hist)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Seeded != 1 {
		t.Fatalf("Seeded = %d, want 1", summary.Seeded)
	}

	visits, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(visits) != 1 {
		t.Fatalf("len(visits) = %d, want 1", len(visits))
	}
	want := info.ModTime().UTC().Truncate(time.Second)
	if !visits[0].At.Equal(want) {
		t.Fatalf("At = %v, want %v", visits[0].At, want)
	}
}

func TestRun_AlreadyJumpedProjectNotOverwritten(t *testing.T) {
	root := t.TempDir()
	newProjectDir(t, root, "tools", "cdd")

	hist := newHistory(t)
	if err := hist.Record("tools/cdd"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	before, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}

	summary, err := scan.Run(context.Background(), newConfig(root, nil), hist)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Seeded != 0 {
		t.Fatalf("Seeded = %d, want 0", summary.Seeded)
	}

	after, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(after) != 1 || after[0].Source != history.SourceJump || !after[0].At.Equal(before[0].At) {
		t.Fatalf("after = %+v, want unchanged jump visit %+v", after, before)
	}
}

func TestRun_SecondRunSeedsNothingNew(t *testing.T) {
	root := t.TempDir()
	newProjectDir(t, root, "tools", "cdd")

	hist := newHistory(t)
	cfg := newConfig(root, nil)

	if _, err := scan.Run(context.Background(), cfg, hist); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	summary, err := scan.Run(context.Background(), cfg, hist)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if summary != (scan.Summary{Seeded: 0, Projects: 1}) {
		t.Fatalf("summary = %+v, want {Seeded:0 Projects:1}", summary)
	}
}

func TestRun_ExcludedProjectSkipped(t *testing.T) {
	root := t.TempDir()
	newProjectDir(t, root, "tools", "cdd")
	newProjectDir(t, root, "tools", "scratch")

	hist := newHistory(t)
	summary, err := scan.Run(context.Background(), newConfig(root, []string{"tools/scratch"}), hist)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary != (scan.Summary{Seeded: 1, Projects: 1}) {
		t.Fatalf("summary = %+v, want {Seeded:1 Projects:1}", summary)
	}

	visits, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(visits) != 1 || visits[0].Project != "tools/cdd" {
		t.Fatalf("visits = %+v, want only tools/cdd", visits)
	}
}
