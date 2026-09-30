package project_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kryft-dev/cdd/internal/project"
)

// mkDirs creates each rel path as a directory tree under root.
func mkDirs(t *testing.T, root string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", rel, err)
		}
	}
}

// mkRepos creates each rel path under root as a Project, with a .git
// directory inside it.
func mkRepos(t *testing.T, root string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		mkDirs(t, root, filepath.Join(rel, ".git"))
	}
}

// discover runs Discover over dirs and fails the test on error.
func discover(t *testing.T, dirs []string, exclude []string, includeHidden bool) []string {
	t.Helper()
	got, err := project.Discover(dirs, exclude, includeHidden)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	return got
}

// assertProjects compares got against the Projects at want's rel paths
// under root, in order.
func assertProjects(t *testing.T, root string, got []string, want ...string) {
	t.Helper()
	abs := make([]string, len(want))
	for i, rel := range want {
		abs[i] = filepath.Join(root, rel)
	}
	if !slices.Equal(got, abs) {
		t.Errorf("Discover() = %v, want %v", got, abs)
	}
}

func TestDiscoverFindsReposAtAnyDepth(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "cdd", "tools/grg", "work/client/api")
	mkDirs(t, root, "notes/2026", "tools/scratch")

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, "cdd", "tools/grg", "work/client/api")
}

func TestDiscoverGitFileCountsAsRepo(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, "worktree")
	if err := os.WriteFile(filepath.Join(root, "worktree", ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, "worktree")
}

func TestDiscoverStopsAtFirstRepo(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "mono", "mono/vendor/lib")

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, "mono")
}

func TestDiscoverWalksIntoGivenDirThatIsRepo(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, ".", "tools/cdd")

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, ".", "tools/cdd")
}

func TestDiscoverManyDirsDeduplicated(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "a/one", "b/two")

	got := discover(t, []string{filepath.Join(root, "a"), filepath.Join(root, "b"), root}, nil, false)
	assertProjects(t, root, got, "a/one", "b/two")
}

func TestDiscoverRelativeDirResolved(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "tools/cdd")
	t.Chdir(root)

	got := discover(t, []string{"tools"}, nil, false)
	assertProjects(t, root, got, "tools/cdd")
}

func TestDiscoverHiddenSkippedByDefault(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, ".dotfiles", ".cache/deep/repo", "visible")

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, "visible")
}

func TestDiscoverIncludeHidden(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, ".dotfiles", ".cache/deep/repo", "visible")

	got := discover(t, []string{root}, nil, true)
	assertProjects(t, root, got, ".cache/deep/repo", ".dotfiles", "visible")
}

func TestDiscoverExcludePatterns(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name    string
		exclude []string
		want    []string
	}{
		{"name alone matches at any depth", []string{"archive"}, []string{"tools/cdd", "tools/scratch"}},
		{"name glob", []string{"scr*"}, []string{"archive/old", "tools/cdd"}},
		{"path pattern", []string{filepath.Join(root, "tools", "*")}, []string{"archive/old"}},
		{"path pattern names one directory", []string{filepath.Join(root, "tools", "scratch")}, []string{"archive/old", "tools/cdd"}},
	}
	mkRepos(t, root, "tools/cdd", "tools/scratch", "archive/old")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discover(t, []string{root}, tt.exclude, false)
			assertProjects(t, root, got, tt.want...)
		})
	}
}

func TestDiscoverInvalidPattern(t *testing.T) {
	if _, err := project.Discover([]string{t.TempDir()}, []string{"["}, false); err == nil {
		t.Fatal("Discover with invalid pattern: want error, got nil")
	}
}

func TestDiscoverMissingDirIsError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := project.Discover([]string{missing}, nil, false); err == nil {
		t.Fatal("Discover of a missing directory: want error, got nil")
	}
}

func TestDiscoverSymlinksNotFollowedDuringWalk(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	mkRepos(t, elsewhere, "real")
	if err := os.Symlink(elsewhere, filepath.Join(root, "link")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got)
}

func TestDiscoverGivenSymlinkIsFollowed(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	mkRepos(t, elsewhere, "real")
	link := filepath.Join(root, "link")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	got := discover(t, []string{link}, nil, false)
	assertProjects(t, link, got, "real")
}

func TestDiscoverUnreadableDirSkipped(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "open/repo")
	mkDirs(t, root, "locked")
	if err := os.Chmod(filepath.Join(root, "locked"), 0); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o755) })

	got := discover(t, []string{root}, nil, false)
	assertProjects(t, root, got, "open/repo")
}
