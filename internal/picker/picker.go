// Package picker is the interactive screen that lists Projects and lets the
// user choose one to Jump to.
//
// The Picker never computes History order itself: Run receives rows already
// ordered by the caller (History order, most recent Visit first) and only
// filters and displays them.
package picker

import (
	"context"
	"errors"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/git"
)

// Project is the identity of one Picker row: its parent directory as shown,
// its name, and its absolute path on disk.
type Project struct {
	// Dir is the Project's parent directory as the row shows it, ending in
	// a separator, with the home directory shortened to "~" (e.g.
	// "~/Developer/tools/").
	Dir string
	// Name is the Project directory's own name, e.g. "cdd".
	Name string
	// Path is the Project's absolute path, where a Jump lands.
	Path string
}

// Row is one line the Picker can show and choose: a Project plus what
// History knows about it.
type Row struct {
	Project Project

	// LastVisit is the time of the Project's most recent Visit, or the zero
	// time when the Project has never been visited.
	LastVisit time.Time

	// Visits is the count of Visits History holds for the Project.
	Visits int
}

// Choice is what the user chose in the Picker: a Project and the Action to
// run on it, the built-in jump for a plain Enter. A detached Action never
// ends up here, since the Picker runs it and stays open.
type Choice struct {
	Row Row

	// Action is the Action to run on the Row. The Picker always sets it; a
	// nil one is taken as a plain Jump.
	Action *action.Action
}

// StatusFunc reports a directory's git status. The Picker calls it once per
// row, concurrently, to fill in the status column and the preview pane
// without blocking the screen.
type StatusFunc func(ctx context.Context, dir string) git.Status

// Layout names one of the Picker's layouts. It is a standing
// preference, set once in config.toml, not a per-session toggle.
type Layout string

// LayoutList is the flat fzf-style layout, and the default: rows in History
// order with the parent directory muted before each Project name, the filter prompt
// below the list, and a bar plus background highlight on the selected row.
const LayoutList Layout = "list"

// Options configures a Run of the Picker.
type Options struct {
	// Vim selects the vim key map: the list is focused on open, j/k move,
	// g/G jump to the ends, f or / focuses the filter, esc in the filter
	// returns to the list keeping the query, esc or q on the list cancels.
	//
	// When false, the default key map applies: typing filters, arrow keys
	// (and ctrl+p/ctrl+n) move, esc cancels, ctrl+u clears the filter.
	Vim bool

	// Query seeds the filter line.
	Query string

	// Layout selects which layout is drawn. The zero value is
	// LayoutList.
	Layout Layout

	// Actions are the Actions bound to keys, as config resolves them. The
	// footer hints list them in this order, the one on enter first.
	Actions []action.Action

	// HideHints leaves the key hints off the footer line, which still
	// shows the match count and any message.
	HideHints bool

	// Runner starts the detached Actions, which leave the Picker open. The
	// zero value is action.ExecRunner.
	Runner action.Runner

	// Copy puts a Project's path on the clipboard for the Action with Copy
	// set. It reports false when no clipboard program is available, and the
	// Picker falls back to the OSC 52 escape. The zero value is action.Copy.
	Copy func(text string) (bool, error)
}

// concurrency bounds how many StatusFunc calls run at once, so a large
// History does not spawn one git process per Project.
const concurrency = 8

// Run draws the Picker over rows, which must already be in History order,
// and lets the user filter and choose one. It returns the Choice and true,
// or the zero Choice and false when the user cancels.
//
// The Picker draws on /dev/tty via tea.OpenTTY, falling back to stderr when
// no TTY is available. It never writes to stdout.
func Run(rows []Row, status StatusFunc, opts Options) (Choice, bool, error) {
	m := NewModel(rows, status, opts)

	ttyOpts, cleanup, err := ttyProgramOptions()
	if err != nil {
		return Choice{}, false, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	p := tea.NewProgram(m, ttyOpts...)
	final, err := p.Run()
	if err != nil {
		return Choice{}, false, err
	}

	fm, ok := final.(Model)
	if !ok {
		return Choice{}, false, errors.New("picker: unexpected Model type from Bubble Tea")
	}
	if !fm.chosen {
		return Choice{}, false, nil
	}
	return Choice{Row: fm.chosenRow, Action: fm.chosenAction}, true, nil
}

// ttyProgramOptions builds the tea.ProgramOptions that make the Picker draw
// on /dev/tty, falling back to stderr when no TTY can be opened. stdout is
// never used, since a caller may pipe it (the Wrapper reads the chosen path
// from Run's return value, not from the program's own output).
func ttyProgramOptions() ([]tea.ProgramOption, func(), error) {
	in, out, err := tea.OpenTTY()
	if err != nil {
		return []tea.ProgramOption{tea.WithOutput(os.Stderr)}, nil, nil
	}
	cleanup := func() {
		_ = in.Close()
		_ = out.Close()
	}
	return []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(out)}, cleanup, nil
}
