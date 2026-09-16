package git_test

import (
	"context"
	"os"
	"testing"

	cdgit "github.com/kryft-dev/cdd/internal/git"
)

func TestGetStatus_Clean(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Kind != cdgit.Found {
		t.Fatalf("Kind = %v, want Found", got.Kind)
	}
	if got.State != cdgit.Clean {
		t.Errorf("State = %v, want Clean", got.State)
	}
	if got.Untracked {
		t.Errorf("Untracked = true, want false")
	}
	if got.Branch != "main" {
		t.Errorf("Branch = %q, want %q", got.Branch, "main")
	}
	if got.HasUpstream {
		t.Errorf("HasUpstream = true, want false")
	}
}

func TestGetStatus_Modified(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")
	writeFile(t, dir, "a.txt", "changed\n")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.State != cdgit.Dirty {
		t.Errorf("State = %v, want Dirty", got.State)
	}
}

func TestGetStatus_UntrackedOnly(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")
	writeFile(t, dir, "b.txt", "new\n")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.State != cdgit.Clean {
		t.Errorf("State = %v, want Clean", got.State)
	}
	if !got.Untracked {
		t.Errorf("Untracked = false, want true")
	}
}

// setupUpstream creates a bare repo, clones it into a work dir, and returns
// the clone's path.
func setupUpstream(t *testing.T) string {
	t.Helper()

	upstream := t.TempDir()
	run(t, upstream, "init", "-q", "--bare")

	work := t.TempDir()
	run(t, work, "clone", "-q", upstream, "clone")
	clone := work + "/clone"
	run(t, clone, "config", "user.name", "cdd test")
	run(t, clone, "config", "user.email", "cdd-test@example.com")
	run(t, clone, "checkout", "-q", "-b", "main")

	writeFile(t, clone, "a.txt", "hello\n")
	commit(t, clone, "initial")
	run(t, clone, "push", "-q", "-u", "origin", "main")

	return clone
}

func TestGetStatus_Ahead(t *testing.T) {
	dir := setupUpstream(t)
	writeFile(t, dir, "b.txt", "new\n")
	commit(t, dir, "second")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if !got.HasUpstream {
		t.Fatalf("HasUpstream = false, want true")
	}
	if got.Ahead != 1 || got.Behind != 0 {
		t.Errorf("Ahead/Behind = %d/%d, want 1/0", got.Ahead, got.Behind)
	}
}

func TestGetStatus_Behind(t *testing.T) {
	dir := setupUpstream(t)

	// A second clone of the same bare upstream pushes a commit, putting dir
	// behind origin/main.
	upstreamPath := trimNL(run(t, dir, "remote", "get-url", "origin"))

	otherClone := t.TempDir()
	run(t, otherClone, "clone", "-q", upstreamPath, "clone2")
	oc := otherClone + "/clone2"
	run(t, oc, "config", "user.name", "cdd test")
	run(t, oc, "config", "user.email", "cdd-test@example.com")
	// The bare upstream's HEAD still points at the default branch name from
	// its own `init`, which may not be "main"; check out main explicitly so
	// this clone builds on top of dir's history rather than starting an
	// unrelated branch.
	run(t, oc, "checkout", "-q", "main")
	writeFile(t, oc, "c.txt", "new\n")
	commit(t, oc, "from other clone")
	run(t, oc, "push", "-q")

	run(t, dir, "fetch", "-q")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Ahead != 0 || got.Behind != 1 {
		t.Errorf("Ahead/Behind = %d/%d, want 0/1", got.Ahead, got.Behind)
	}
}

func TestGetStatus_NoUpstream(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.HasUpstream {
		t.Errorf("HasUpstream = true, want false")
	}
	if got.Ahead != 0 || got.Behind != 0 {
		t.Errorf("Ahead/Behind = %d/%d, want 0/0", got.Ahead, got.Behind)
	}
}

func TestGetStatus_DetachedHEAD(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")
	writeFile(t, dir, "b.txt", "second\n")
	commit(t, dir, "second")
	run(t, dir, "checkout", "-q", "HEAD~1")

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Branch != "(detached)" {
		t.Errorf("Branch = %q, want %q", got.Branch, "(detached)")
	}
	if got.HasUpstream {
		t.Errorf("HasUpstream = true, want false")
	}
}

func TestGetStatus_Unborn(t *testing.T) {
	dir := newRepo(t)

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Kind != cdgit.Found {
		t.Fatalf("Kind = %v, want Found", got.Kind)
	}
	if got.State != cdgit.Clean {
		t.Errorf("State = %v, want Clean", got.State)
	}
}

func TestGetStatus_NotRepo(t *testing.T) {
	dir := t.TempDir()

	got, err := cdgit.GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Kind != cdgit.NotRepo {
		t.Fatalf("Kind = %v, want NotRepo", got.Kind)
	}
}

func TestGetStatus_SubdirectoryOfRepoIsNotRepo(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")

	sub := dir + "/sub"
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	got, err := cdgit.GetStatus(context.Background(), sub)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Kind != cdgit.NotRepo {
		t.Fatalf("Kind = %v, want NotRepo (git must not walk up to the enclosing repo)", got.Kind)
	}
}

func TestGetStatus_Timeout(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := cdgit.GetStatus(ctx, dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if got.Kind != cdgit.Unknown {
		t.Fatalf("Kind = %v, want Unknown (a cancelled context must never report Clean)", got.Kind)
	}
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
