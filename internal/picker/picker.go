// Package picker is the interactive screen that lists Projects and lets the
// user choose one to Jump to.
//
// The Picker never computes History order itself: Run receives rows already
// ordered by the caller (History order, most recent Visit first, then
// never-visited Projects A-Z) and only filters and displays them.
package picker

import (
	"context"
	"errors"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/git"
)

// Project is the identity of one Picker row: its Kind, its Name, and its
// absolute path on disk.
type Project struct {
	Kind string
	Name string
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

// StatusFunc reports a directory's git status. The Picker calls it once per
// row, concurrently, to fill in the status column and the preview pane
// without blocking the screen.
type StatusFunc func(ctx context.Context, dir string) git.Status

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
}

// concurrency bounds how many StatusFunc calls run at once, so a large
// History does not spawn one git process per Project.
const concurrency = 8

// Run draws the Picker over rows, which must already be in History order,
// and lets the user filter and choose one. It returns the chosen Row and
// true, or the zero Row and false when the user cancels.
//
// The Picker draws on /dev/tty via tea.OpenTTY, falling back to stderr when
// no TTY is available. It never writes to stdout.
func Run(rows []Row, status StatusFunc, opts Options) (Row, bool, error) {
	m := NewModel(rows, status, opts)

	ttyOpts, cleanup, err := ttyProgramOptions()
	if err != nil {
		return Row{}, false, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	p := tea.NewProgram(m, ttyOpts...)
	final, err := p.Run()
	if err != nil {
		return Row{}, false, err
	}

	fm, ok := final.(Model)
	if !ok {
		return Row{}, false, errors.New("picker: unexpected Model type from Bubble Tea")
	}
	if !fm.chosen {
		return Row{}, false, nil
	}
	return fm.chosenRow, true, nil
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
