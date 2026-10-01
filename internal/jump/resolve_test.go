package jump_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
)

// mkProjects creates each rel path under a fresh root as a Project (a
// directory holding .git) and Records a Visit for it in hist, in order, so
// the last one is the most recent. It returns a Config and the root.
func mkProjects(t *testing.T, hist *history.History, rels ...string) (config.Config, string) {
	t.Helper()

	root := t.TempDir()
	for _, rel := range rels {
		if err := os.MkdirAll(filepath.Join(root, rel, ".git"), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", rel, err)
		}
		if err := hist.Record(filepath.Join(root, rel)); err != nil {
			t.Fatalf("Record(%q): %v", rel, err)
		}
	}

	return config.Config{History: config.History{MaxVisits: 1000}}, root
}

// capturePick returns a PickFunc that stores the rows and Options it was
// run with and chooses the row at path, or cancels when path is "".
func capturePick(rows *[]picker.Row, opts *picker.Options, path string) jump.PickFunc {
	return func(r []picker.Row, _ picker.StatusFunc, o picker.Options) (picker.Row, bool, error) {
		*rows, *opts = r, o
		if path == "" {
			return picker.Row{}, false, nil
		}
		return picker.Row{Project: picker.Project{Path: path}}, true, nil
	}
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

func TestResolve_ExactMatchShortcutSkipsPicker(t *testing.T) {
	tests := []struct {
		name, query, want string
	}{
		{"name", "grg", "tools/grg"},
		{"trailing path", "tools/cdd", "tools/cdd"},
		{"trailing path with slash", "tools/cdd/", "tools/cdd"},
		{"deeper trailing path", "work/client/cdd", "work/client/cdd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hist := newHistory(t)
			cfg, root := mkProjects(t, hist, "tools/cdd", "tools/grg", "work/client/cdd")

			got, err := jump.Resolve(context.Background(), cfg, hist, tt.query, failPick(t))
			if err != nil {
				t.Fatalf("Resolve: unexpected error: %v", err)
			}
			if want := filepath.Join(root, tt.want); got != want {
				t.Errorf("Resolve = %q, want %q", got, want)
			}
		})
	}
}

func TestResolve_WholePathShortcutSkipsPicker(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	want := filepath.Join(root, "tools", "cdd")

	got, err := jump.Resolve(context.Background(), cfg, hist, want, failPick(t))
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

func TestResolve_AmbiguousOrPartialQueryOpensPickerPrefilled(t *testing.T) {
	for _, query := range []string{"cdd", "ools/cdd", "nope"} {
		t.Run(query, func(t *testing.T) {
			hist := newHistory(t)
			cfg, root := mkProjects(t, hist, "archive/cdd", "tools/cdd")
			want := filepath.Join(root, "tools", "cdd")

			var rows []picker.Row
			var opts picker.Options
			got, err := jump.Resolve(context.Background(), cfg, hist, query, capturePick(&rows, &opts, want))
			if err != nil {
				t.Fatalf("Resolve: unexpected error: %v", err)
			}

			if opts.Query != query {
				t.Errorf("Picker Query = %q, want %q", opts.Query, query)
			}
			if len(rows) != 2 {
				t.Errorf("Picker got %d rows, want 2", len(rows))
			}
			if got != want {
				t.Errorf("Resolve = %q, want %q", got, want)
			}
			assertRecorded(t, hist, want)
		})
	}
}

func TestResolve_RowsComeFromHistoryNewestFirst(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "old")
	if err := os.MkdirAll(filepath.Join(root, "never-visited", ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	newer := filepath.Join(root, "newer")
	if err := os.MkdirAll(filepath.Join(newer, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := hist.Seed(newer, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	var rows []picker.Row
	var opts picker.Options
	_, _ = jump.Resolve(context.Background(), cfg, hist, "", capturePick(&rows, &opts, ""))

	var got []string
	for _, r := range rows {
		got = append(got, r.Project.Path)
	}
	want := []string{newer, filepath.Join(root, "old")}
	if !slices.Equal(got, want) {
		t.Errorf("Picker rows = %v, want %v (History only, newest first)", got, want)
	}
}

func TestResolve_StaleVisitGivesNoRowAndNoShortcut(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd", "archive/cdd")
	if err := os.RemoveAll(filepath.Join(root, "archive", "cdd", ".git")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	got, err := jump.Resolve(context.Background(), cfg, hist, "cdd", failPick(t))
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if want := filepath.Join(root, "tools", "cdd"); got != want {
		t.Errorf("Resolve = %q, want %q (the Stale Visit leaves one match)", got, want)
	}

	var rows []picker.Row
	var opts picker.Options
	_, _ = jump.Resolve(context.Background(), cfg, hist, "", capturePick(&rows, &opts, ""))
	if len(rows) != 1 {
		t.Errorf("Picker got %d rows, want 1 (Stale Visit dropped)", len(rows))
	}
}

// TestResolve_ForwardsPickerOptions verifies that the config keys that
// configure the Picker reach it: the vim key map and the layout.
func TestResolve_ForwardsPickerOptions(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	cfg.Keys.Vim = true
	cfg.Picker.Layout = "list"

	var rows []picker.Row
	var got picker.Options
	if _, err := jump.Resolve(context.Background(), cfg, hist, "", capturePick(&rows, &got, filepath.Join(root, "tools", "cdd"))); err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}

	if !got.Vim {
		t.Errorf("Options.Vim = false, want true")
	}
	if got.Layout != picker.LayoutList {
		t.Errorf("Options.Layout = %q, want %q", got.Layout, picker.LayoutList)
	}
}

func TestResolve_CancelReturnsErrCancelled(t *testing.T) {
	hist := newHistory(t)
	cfg, _ := mkProjects(t, hist, "tools/cdd")

	var rows []picker.Row
	var opts picker.Options
	_, err := jump.Resolve(context.Background(), cfg, hist, "anything", capturePick(&rows, &opts, ""))
	if !errors.Is(err, jump.ErrCancelled) {
		t.Fatalf("Resolve error = %v, want ErrCancelled", err)
	}
}

func TestResolve_VanishedDirectoryErrors(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")

	var rows []picker.Row
	var opts picker.Options
	gone := filepath.Join(root, "tools", "vanished")
	if _, err := jump.Resolve(context.Background(), cfg, hist, "anything", capturePick(&rows, &opts, gone)); err == nil {
		t.Fatal("Resolve: want error for a chosen directory that no longer exists, got nil")
	}
}

func TestResolve_FailingHistoryWriteStillReturnsPath(t *testing.T) {
	historyDir := t.TempDir()
	historyPath := filepath.Join(historyDir, "history")
	hist, err := history.Open(historyPath, 1000)
	if err != nil {
		t.Fatalf("history.Open: %v", err)
	}
	cfg, root := mkProjects(t, hist, "tools/cdd")

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

// assertRecorded checks History's latest Visits contain path.
func assertRecorded(t *testing.T, hist *history.History, path string) {
	t.Helper()

	latest, err := hist.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	for _, v := range latest {
		if v.Project == path {
			return
		}
	}
	t.Errorf("History has no Visit for %q after Resolve", path)
}

// TestResolve_WordQueryOpensPicker checks that a query with a space is
// always a parent-then-name filter for the Picker, never the exact-match
// shortcut, even when a Project's path happens to end in it.
func TestResolve_WordQueryOpensPicker(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/my cdd")
	want := filepath.Join(root, "tools", "my cdd")

	var rows []picker.Row
	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, "my cdd", capturePick(&rows, &opts, want))
	if err != nil {
		t.Fatalf("Resolve: unexpected error: %v", err)
	}
	if opts.Query != "my cdd" {
		t.Errorf("Picker Query = %q, want %q (the shortcut skipped the Picker)", opts.Query, "my cdd")
	}
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}
