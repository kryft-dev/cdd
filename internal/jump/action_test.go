package jump_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
)

// fakeRunner stands in for the Runner, so no test launches a program. It
// records what ran, and answers Run with code and runErr, Start with
// startErr.
type fakeRunner struct {
	ran      []string // "name path"
	code     int
	runErr   error
	startErr error
}

func (f *fakeRunner) Start(a action.Action, path string) error {
	f.ran = append(f.ran, "start "+a.Name+" "+path)
	return f.startErr
}

func (f *fakeRunner) Run(a action.Action, path string) (int, error) {
	f.ran = append(f.ran, "run "+a.Name+" "+path)
	return f.code, f.runErr
}

// pickWith returns a PickFunc that chooses path with a, and leaves the
// Options it was given in opts.
func pickWith(a action.Action, path string, opts *picker.Options) jump.PickFunc {
	return func(_ []picker.Row, _ picker.StatusFunc, o picker.Options) (picker.Choice, bool, error) {
		*opts = o
		return picker.Choice{Row: picker.Row{Project: picker.Project{Path: path}}, Action: &a}, true, nil
	}
}

func visitCount(t *testing.T, hist *history.History, path string) int {
	t.Helper()
	n, err := hist.Count(path)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	return n
}

func TestResolve_AttachedActionRunsOnTheTerminalAndRecordsAVisit(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")
	before := visitCount(t, hist, path)

	run := &fakeRunner{}
	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "lazygit", Run: "lazygit"}, path, &opts), run)

	if err != nil || got != "" {
		t.Errorf("Resolve = %q, %v, want no path and no error", got, err)
	}
	if len(run.ran) != 1 || run.ran[0] != "run lazygit "+path {
		t.Errorf("ran = %v, want lazygit on %s", run.ran, path)
	}
	if n := visitCount(t, hist, path); n != before+1 {
		t.Errorf("Visits = %d, want %d", n, before+1)
	}
}

func TestResolve_JumpActionReturnsThePathAfterTheCommand(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")

	run := &fakeRunner{}
	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "fmt", Run: "gofmt", Jump: true}, path, &opts), run)

	if err != nil || got != path {
		t.Errorf("Resolve = %q, %v, want %q", got, err, path)
	}
	if len(run.ran) != 1 {
		t.Errorf("ran = %v, want the command to have run", run.ran)
	}
}

func TestResolve_ActionWithoutACommandOnlyJumps(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")

	run := &fakeRunner{}
	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "jump", Jump: true}, path, &opts), run)

	if err != nil || got != path || len(run.ran) != 0 {
		t.Errorf("Resolve = %q, %v, ran %v, want the path and no command", got, err, run.ran)
	}
}

func TestResolve_FailingCommandReturnsItsExitStatusAndNoPath(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")

	var opts picker.Options
	got, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "x", Run: "x", Jump: true}, path, &opts), &fakeRunner{code: 3})

	var exit *jump.ExitError
	if !errors.As(err, &exit) || exit.Code != 3 || got != "" {
		t.Errorf("Resolve = %q, %v, want ExitError{3} and no path", got, err)
	}
}

func TestResolve_CommandThatCannotRunIsAnError(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")

	var opts picker.Options
	_, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "x", Run: "x"}, path, &opts), &fakeRunner{runErr: errors.New("no tty")})

	var exit *jump.ExitError
	if err == nil || errors.As(err, &exit) {
		t.Errorf("Resolve error = %v, want a plain error", err)
	}
}

func TestResolve_PickerGetsTheResolvedActionsAndARecordingRunner(t *testing.T) {
	hist := newHistory(t)
	cfg, root := mkProjects(t, hist, "tools/cdd")
	path := filepath.Join(root, "tools", "cdd")
	cfg.ResolvedActions = []action.Action{{Name: "code", Key: "ctrl+v", Run: "code", Detach: true}}

	run := &fakeRunner{}
	var opts picker.Options
	if _, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "jump", Jump: true}, path, &opts), run); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(opts.Actions) != 1 || opts.Actions[0].Name != "code" {
		t.Errorf("Options.Actions = %v, want the config's resolved Actions", opts.Actions)
	}

	// What the Picker does when a detached Action key is pressed.
	before := visitCount(t, hist, path)
	if err := opts.Runner.Start(opts.Actions[0], path); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if n := visitCount(t, hist, path); n != before+1 {
		t.Errorf("Visits = %d, want %d after a detached Action", n, before+1)
	}

	// A detached Action that fails to start is no Visit.
	run.startErr = errors.New("boom")
	before = visitCount(t, hist, path)
	if err := opts.Runner.Start(opts.Actions[0], path); err == nil {
		t.Fatal("Start: want the runner's error")
	}
	if n := visitCount(t, hist, path); n != before {
		t.Errorf("Visits = %d, want %d after a failed start", n, before)
	}
}

func TestResolve_PickerHidesHintsWhenConfigSaysSo(t *testing.T) {
	for _, hints := range []bool{true, false} {
		hist := newHistory(t)
		cfg, root := mkProjects(t, hist, "tools/cdd")
		path := filepath.Join(root, "tools", "cdd")
		cfg.Picker.Hints = hints

		var opts picker.Options
		if _, err := jump.Resolve(context.Background(), cfg, hist, pickWith(action.Action{Name: "jump", Jump: true}, path, &opts), &fakeRunner{}); err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if opts.HideHints == hints {
			t.Errorf("Picker.Hints = %v: Options.HideHints = %v", hints, opts.HideHints)
		}
	}
}
