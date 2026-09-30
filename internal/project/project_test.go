package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/project"
)

func TestIsRepo(t *testing.T) {
	root := t.TempDir()
	mkRepos(t, root, "clone")
	mkDirs(t, root, "plain", "worktree")
	if err := os.WriteFile(filepath.Join(root, "worktree", ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	for dir, want := range map[string]bool{"clone": true, "worktree": true, "plain": false, "missing": false} {
		if got := project.IsRepo(filepath.Join(root, dir)); got != want {
			t.Errorf("IsRepo(%s) = %v, want %v", dir, got, want)
		}
	}
}
