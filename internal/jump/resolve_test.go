package jump_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
)

// mkProjects creates each rel path as a Kind/Name directory tree under
// root, and returns a Config pointing at root.
func mkProjects(t *testing.T, rels ...string) (config.Config, string) {
	t.Helper()

	root := t.TempDir()
	for _, rel := range rels {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", rel, err)
		}
	}

	return config.Config{Root: root, History: config.History{MaxVisits: 1000}}, root
}

// newHistory opens a fresh History in a temp directory.
func newHistory(t *testing.T) *history.History {
	t.Helper()

	path := filepath.Join(t.TempDir(), "history")
	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("history.Open: %v", err)
	}
	return h
}

// failPick fails the test if the Picker is ever run; it is used by tests
// that expect the exact-match shortcut to skip it.
func failPick(t *testing.T) jump.PickFunc {
	t.Helper()
	return func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error) {
		t.Fatal("pick: Picker was run, want the exact-match shortcut to skip it")
		return picker.Row{}, false, nil
	}
}

func TestResolve_ExactNameShortcutSkipsPicker(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd")
	hist := newHistory(t)

	got, err := jump.Resolve(context.Background(), cfg, hist, "cdd", failPick(t))
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}

	want := filepath.Join(root, "tools", "cdd")
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}

	assertRecorded(t, hist, "tools/cdd")
}

func TestResolve_ExactKindNameShortcutSkipsPicker(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd", "archive/cdd")
	hist := newHistory(t)

	got, err := jump.Resolve(context.Background(), cfg, hist, "tools/cdd", failPick(t))
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}

	want := filepath.Join(root, "tools", "cdd")
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}

	assertRecorded(t, hist, "tools/cdd")
}

func TestResolve_AmbiguousNameOpensPickerPrefilled(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd", "archive/cdd")
	hist := newHistory(t)

	var gotQuery string
	var gotRows int
	pick := func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error) {
		gotQuery = opts.Query
		gotRows = len(rows)
		return picker.Row{Project: picker.Project{
			Kind: "tools", Name: "cdd", Path: filepath.Join(root, "tools", "cdd"),
		}}, true, nil
	}

	got, err := jump.Resolve(context.Background(), cfg, hist, "cdd", pick)
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}

	if gotQuery != "cdd" {
		t.Errorf("Picker Query = %q, want %q", gotQuery, "cdd")
	}
	if gotRows != 2 {
		t.Errorf("Picker got %d rows, want 2 (both Projects named cdd)", gotRows)
	}

	want := filepath.Join(root, "tools", "cdd")
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}

	assertRecorded(t, hist, "tools/cdd")
}

func TestResolve_NoMatchOpensPickerPrefilled(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd")
	hist := newHistory(t)

	var gotQuery string
	pick := func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error) {
		gotQuery = opts.Query
		return picker.Row{Project: picker.Project{
			Kind: "tools", Name: "cdd", Path: filepath.Join(root, "tools", "cdd"),
		}}, true, nil
	}

	if _, err := jump.Resolve(context.Background(), cfg, hist, "nope", pick); err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if gotQuery != "nope" {
		t.Errorf("Picker Query = %q, want %q", gotQuery, "nope")
	}
}

func TestResolve_CancelReturnsErrCancelled(t *testing.T) {
	cfg, _ := mkProjects(t, "tools/cdd")
	hist := newHistory(t)

	pick := func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error) {
		return picker.Row{}, false, nil
	}

	_, err := jump.Resolve(context.Background(), cfg, hist, "anything", pick)
	if !errors.Is(err, jump.ErrCancelled) {
		t.Fatalf("Resolve error = %v, want ErrCancelled", err)
	}
}

func TestResolve_VanishedDirectoryErrors(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd")
	hist := newHistory(t)

	gone := filepath.Join(root, "tools", "vanished")
	pick := func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error) {
		return picker.Row{Project: picker.Project{Kind: "tools", Name: "vanished", Path: gone}}, true, nil
	}

	_, err := jump.Resolve(context.Background(), cfg, hist, "anything", pick)
	if err == nil {
		t.Fatal("Resolve: want error for a chosen directory that no longer exists, got nil")
	}
}

func TestResolve_FailingHistoryWriteStillReturnsPath(t *testing.T) {
	cfg, root := mkProjects(t, "tools/cdd")

	historyDir := t.TempDir()
	historyPath := filepath.Join(historyDir, "history")
	hist, err := history.Open(historyPath, 1000)
	if err != nil {
		t.Fatalf("history.Open: %v", err)
	}

	// Force Record to fail by making its directory unwritable, without
	// touching the Project directories under root.
	if err := os.Chmod(historyDir, 0o500); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(historyDir, 0o700) })

	got, err := jump.Resolve(context.Background(), cfg, hist, "cdd", failPick(t))
	if err != nil {
		t.Fatalf("Resolve: unexpected error despite a failing History write: %v", err)
	}

	want := filepath.Join(root, "tools", "cdd")
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

// assertRecorded checks History's latest Visits contain rel.
func assertRecorded(t *testing.T, hist *history.History, rel string) {
	t.Helper()

	latest, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	for _, v := range latest {
		if v.Project == rel {
			return
		}
	}
	t.Errorf("History has no Visit for %q after Resolve", rel)
}
