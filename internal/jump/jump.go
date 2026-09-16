package jump

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

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
// the Project to Jump to: discover Projects, read History's latest Visits,
// take the exact-match shortcut on a Project's Name or Rel, otherwise run
// the Picker (via pick) with query prefilled, confirm the chosen
// directory still exists, Record the Visit, and return the absolute path.
//
// A cancelled Picker yields ErrCancelled. A Visit that fails to Record
// only prints a warning to stderr; Resolve still returns the path.
func Resolve(ctx context.Context, cfg config.Config, hist *history.History, query string, pick PickFunc) (string, error) {
	projects, err := project.Discover(cfg.Root, cfg.Exclude, cfg.IncludeHidden)
	if err != nil {
		return "", fmt.Errorf("jump: discover projects: %w", err)
	}

	latest, err := hist.Latest()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	rel, abs, err := choose(cfg, projects, latest, query, pick)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("jump: %q no longer exists: %w", abs, err)
	}

	if err := hist.Record(rel); err != nil {
		fmt.Fprintf(os.Stderr, "cdd: warning: recording Visit for %q: %v\n", rel, err)
	}

	return abs, nil
}

// choose picks a Project either via the exact-match shortcut or by running
// the Picker, and returns its Rel and absolute path.
func choose(cfg config.Config, projects []project.Project, latest []history.Visit, query string, pick PickFunc) (rel, abs string, err error) {
	if p, ok := exactMatch(projects, query); ok {
		return p.Rel(), p.Abs(cfg.Root), nil
	}

	rows := order(projects, latest, cfg.Root)
	status := func(c context.Context, dir string) git.Status {
		s, _ := git.GetStatus(c, dir)
		return s
	}

	row, ok, err := pick(rows, status, picker.Options{Vim: cfg.Keys.Vim, Query: query})
	if err != nil {
		return "", "", fmt.Errorf("jump: %w", err)
	}
	if !ok {
		return "", "", ErrCancelled
	}

	rel = path.Join(row.Project.Kind, row.Project.Name)
	return rel, row.Project.Path, nil
}

// exactMatch reports whether query is exactly one Project's Name or Rel.
// A query matching two or more Projects (e.g. the same Name in different
// Kinds), or none, is not an exact match.
func exactMatch(projects []project.Project, query string) (project.Project, bool) {
	if query == "" {
		return project.Project{}, false
	}

	var found project.Project
	count := 0
	for _, p := range projects {
		if p.Name == query || p.Rel() == query {
			found = p
			count++
		}
	}
	return found, count == 1
}
