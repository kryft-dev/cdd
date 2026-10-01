package jump_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// mkDir creates a plain directory, no .git, at rel under root.
func mkDir(t *testing.T, root, rel string) string {
	t.Helper()

	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", rel, err)
	}
	return dir
}

// shown runs Resolve with a Picker that cancels and returns the paths of
// the rows it was given.
func shown(t *testing.T, hist *history.History, store *project.Store) []string {
	t.Helper()

	cfg, _ := mkProjects(t, hist)
	var rows []picker.Row
	var opts picker.Options
	_, _ = jump.Resolve(context.Background(), cfg, hist, store, capturePick(&rows, &opts, ""), &fakeRunner{})

	var paths []string
	for _, r := range rows {
		paths = append(paths, r.Project.Path)
	}
	return paths
}

func TestResolve_AddedDirectoryWithVisitIsListed(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	dir := mkDir(t, t.TempDir(), "notes")
	if err := store.Add(dir); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := hist.Record(dir); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if got := shown(t, hist, store); !slices.Equal(got, []string{dir}) {
		t.Errorf("rows = %v, want [%s]", got, dir)
	}
}

func TestResolve_VisitedPlainDirectoryNotAddedIsStale(t *testing.T) {
	hist := newHistory(t)
	dir := mkDir(t, t.TempDir(), "notes")
	if err := hist.Record(dir); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if got := shown(t, hist, newStore(t)); len(got) != 0 {
		t.Errorf("rows = %v, want none", got)
	}
}

func TestResolve_AddedDirectoryWithoutVisitsComesAfterVisited(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	_, root := mkProjects(t, hist, "repo")
	b, a := mkDir(t, root, "b"), mkDir(t, root, "a")
	for _, d := range []string{b, a} {
		if err := store.Add(d); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	want := []string{filepath.Join(root, "repo"), a, b}
	if got := shown(t, hist, store); !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestResolve_AddedDirectoryIsStaleOnceGone(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	dir := mkDir(t, t.TempDir(), "notes")
	if err := store.Add(dir); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := hist.Record(dir); err != nil {
		t.Fatalf("Record: %v", err)
	}
	other := mkDir(t, t.TempDir(), "never-visited")
	if err := store.Add(other); err != nil {
		t.Fatalf("Add: %v", err)
	}
	for _, d := range []string{dir, other} {
		if err := os.Remove(d); err != nil {
			t.Fatalf("Remove: %v", err)
		}
	}

	if got := shown(t, hist, store); len(got) != 0 {
		t.Errorf("rows = %v, want none", got)
	}
}

func TestResolve_ForgottenRepositoryIsHidden(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	_, root := mkProjects(t, hist, "gone", "kept")
	if err := store.Forget(filepath.Join(root, "gone")); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	if got, want := shown(t, hist, store), []string{filepath.Join(root, "kept")}; !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
}

func TestResolve_ReAddedRepositoryIsListedAgain(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	_, root := mkProjects(t, hist, "back")
	dir := filepath.Join(root, "back")
	if err := store.Forget(dir); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if err := store.Add(dir); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if got := shown(t, hist, store); !slices.Equal(got, []string{dir}) {
		t.Errorf("rows = %v, want [%s]", got, dir)
	}
}

func TestResolve_AddedDirectoryJumps(t *testing.T) {
	hist := newHistory(t)
	store := newStore(t)
	cfg, root := mkProjects(t, hist)
	dir := mkDir(t, root, "notes")
	if err := store.Add(dir); err != nil {
		t.Fatalf("Add: %v", err)
	}

	var rows []picker.Row
	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, store, capturePick(&rows, &opts, dir), &fakeRunner{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != dir {
		t.Errorf("Resolve = %q, want %q", got, dir)
	}
	assertRecorded(t, hist, dir)
}
