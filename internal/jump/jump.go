package jump

import (
	"context"
	"errors"
	"fmt"
	"os"

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

// Resolve runs the pick flow that turns History into the absolute path of
// the Project to Jump to: read its latest Visits, drop the Stale ones, run
// the Picker (via pick) with an empty Query, confirm the chosen directory
// still exists, Record the Visit, and return the path.
//
// A cancelled Picker yields ErrCancelled. A Visit that fails to Record
// only prints a warning to stderr; Resolve still returns the path.
func Resolve(ctx context.Context, cfg config.Config, hist *history.History, pick PickFunc) (string, error) {
	latest, err := hist.Latest()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	counts, err := hist.Counts()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	path, err := choose(cfg, live(latest), counts, pick)
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

// choose runs the Picker over latest and returns the absolute path of the
// Project it chose.
func choose(cfg config.Config, latest []history.Visit, counts map[string]int, pick PickFunc) (string, error) {
	home, _ := os.UserHomeDir()
	rows := toRows(latest, counts, home)
	status := func(c context.Context, dir string) git.Status {
		s, _ := git.GetStatus(c, dir)
		return s
	}

	row, ok, err := pick(rows, status, picker.Options{Vim: cfg.Keys.Vim, Layout: picker.Layout(cfg.Picker.Layout)})
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}
	if !ok {
		return "", ErrCancelled
	}
	return row.Project.Path, nil
}
