package jump

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// ErrCancelled is returned by Resolve when the Picker is cancelled (Esc,
// Ctrl-C, or q on an empty filter).
var ErrCancelled = errors.New("jump: cancelled")

// PickFunc runs the Picker over rows and returns the chosen Row, mirroring
// picker.Run's signature so tests can inject a fake Picker; production
// passes picker.Run itself.
type PickFunc func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Row, bool, error)

// Resolve runs the pick flow that turns query into the absolute path of
// the Project to Jump to: read History's latest Visits, drop the Stale
// ones, take the exact-match shortcut on a Project's name or trailing path,
// otherwise run the Picker (via pick) with query prefilled, confirm the
// chosen directory still exists, Record the Visit, and return the path.
//
// A cancelled Picker yields ErrCancelled. A Visit that fails to Record
// only prints a warning to stderr; Resolve still returns the path.
func Resolve(ctx context.Context, cfg config.Config, hist *history.History, query string, pick PickFunc) (string, error) {
	latest, err := hist.Latest()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	counts, err := hist.Counts()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	path, err := choose(cfg, live(latest), counts, query, pick)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("jump: %q no longer exists: %w", path, err)
	}

	if err := hist.Record(path); err != nil {
		fmt.Fprintf(os.Stderr, "cdd: warning: recording Visit for %q: %v\n", path, err)
	}

	return path, nil
}

// live drops each Stale Visit from latest: one whose Project no longer
// holds a git repository.
func live(latest []history.Visit) []history.Visit {
	out := make([]history.Visit, 0, len(latest))
	for _, v := range latest {
		if project.IsRepo(v.Project) {
			out = append(out, v)
		}
	}
	return out
}

// choose picks a Project either via the exact-match shortcut or by running
// the Picker, and returns its absolute path.
func choose(cfg config.Config, latest []history.Visit, counts map[string]int, query string, pick PickFunc) (string, error) {
	if p, ok := exactMatch(latest, query); ok {
		return p, nil
	}

	home, _ := os.UserHomeDir()
	rows := toRows(latest, counts, home)
	status := func(c context.Context, dir string) git.Status {
		s, _ := git.GetStatus(c, dir)
		return s
	}

	row, ok, err := pick(rows, status, picker.Options{Vim: cfg.Keys.Vim, Query: query, Layout: picker.LayoutStyle(cfg.Picker.Layout)})
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}
	if !ok {
		return "", ErrCancelled
	}
	return row.Project.Path, nil
}

// exactMatch reports whether query names exactly one Project in latest:
// its name, or any trailing run of its path ("cdd", "tools/cdd"), or the
// whole path. A query matching two or more Projects, or none, is not an
// exact match, and neither is one holding a space: that is a parent-then-
// name filter for the Picker.
func exactMatch(latest []history.Visit, query string) (string, bool) {
	query = strings.TrimSuffix(query, "/")
	if query == "" || strings.ContainsRune(query, ' ') {
		return "", false
	}

	var found string
	count := 0
	for _, v := range latest {
		if v.Project == query || strings.HasSuffix(v.Project, string(filepath.Separator)+query) {
			found = v.Project
			count++
		}
	}
	return found, count == 1
}
