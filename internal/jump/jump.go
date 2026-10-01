package jump

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// ErrCancelled is returned by Resolve when the Picker is cancelled (Esc,
// Ctrl-C, or q on an empty filter).
var ErrCancelled = errors.New("jump: cancelled")

// ExitError is returned by Resolve when an Action's command exits non-zero.
// The command has already said what went wrong on the terminal, so the CLI
// exits with Code and prints nothing.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("jump: action exited with status %d", e.Code) }

// PickFunc runs the Picker over rows and returns the Choice, mirroring
// picker.Run's signature so tests can inject a fake Picker; production
// passes picker.Run itself.
type PickFunc func(rows []picker.Row, status picker.StatusFunc, opts picker.Options) (picker.Choice, bool, error)

// Resolve runs the pick flow that turns History into the absolute path of
// the Project to Jump to: read its latest Visits, drop the Stale and
// Forgotten ones, list the Added Projects that have no Visit last, run the
// Picker (via pick), confirm the chosen directory still exists, Record
// the Visit, and return the path.
//
// When the user ran an Action that does not detach, Resolve also runs its
// command on the terminal through run, once the Picker has quit. It
// returns the path only if the Action Jumps, "" if not, and an *ExitError
// if the command exited non-zero. A detached Action runs inside the Picker
// through run too, Recording its Visit.
//
// A cancelled Picker yields ErrCancelled. A Visit that fails to Record
// only prints a warning to stderr; Resolve still returns the path.
func Resolve(ctx context.Context, cfg config.Config, hist *history.History, store *project.Store, pick PickFunc, run action.Runner) (string, error) {
	latest, err := hist.Latest()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	marks, err := store.Marks()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	counts, err := hist.Counts()
	if err != nil {
		return "", fmt.Errorf("jump: %w", err)
	}

	choice, err := choose(cfg, listed(latest, marks), counts, pick, recorder{run, hist})
	if err != nil {
		return "", err
	}
	path := choice.Row.Project.Path

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("jump: %q no longer exists: %w", path, err)
	}

	record(hist, path)

	a := choice.Action
	if a == nil {
		return path, nil
	}
	if a.Run != "" {
		code, err := run.Run(*a, path)
		if err != nil {
			return "", fmt.Errorf("jump: run %s: %w", a.Name, err)
		}
		if code != 0 {
			return "", &ExitError{code}
		}
	}
	if a.Jump {
		return path, nil
	}
	return "", nil
}

// record Records a Visit for path, warning on stderr when it cannot.
func record(hist *history.History, path string) {
	if err := hist.Record(path); err != nil {
		fmt.Fprintf(os.Stderr, "cdd: warning: recording Visit for %q: %v\n", path, err)
	}
}

// recorder is the Runner the Picker gets: it Records a Visit for each
// detached Action that starts.
type recorder struct {
	action.Runner
	hist *history.History
}

func (r recorder) Start(a action.Action, path string) error {
	if err := r.Runner.Start(a, path); err != nil {
		return err
	}
	record(r.hist, path)
	return nil
}

// choose runs the Picker over latest, starting its detached Actions with
// run, and returns what it chose.
func choose(cfg config.Config, latest []history.Visit, counts map[string]int, pick PickFunc, run action.Runner) (picker.Choice, error) {
	home, _ := os.UserHomeDir()
	rows := toRows(latest, counts, home)
	status := func(c context.Context, dir string) git.Status {
		s, _ := git.GetStatus(c, dir)
		return s
	}

	opts := picker.Options{
		Vim:       cfg.Keys.Vim,
		Layout:    picker.Layout(cfg.Picker.Layout),
		Actions:   cfg.ResolvedActions,
		HideHints: !cfg.Picker.Hints,
		Runner:    run,
	}
	choice, ok, err := pick(rows, status, opts)
	if err != nil {
		return picker.Choice{}, fmt.Errorf("jump: %w", err)
	}
	if !ok {
		return picker.Choice{}, ErrCancelled
	}
	return choice, nil
}
