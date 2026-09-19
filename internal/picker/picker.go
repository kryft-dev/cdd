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
	"charm.land/lipgloss/v2"

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

// LayoutStyle names one of the Picker's two layouts. It is a standing
// preference, set once in config.toml, not a per-session toggle.
type LayoutStyle string

const (
	// LayoutGrouped is the default layout: rows grouped under Kind
	// headers, a caret on the selected row, and the filter line on top.
	LayoutGrouped LayoutStyle = "grouped"

	// LayoutList is the flat fzf-style layout: no Kind headers, rows in
	// History order with "kind/" muted before each Project name, the
	// filter prompt below the list, and a bar plus background highlight
	// on the selected row.
	LayoutList LayoutStyle = "list"
)

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
	// LayoutGrouped.
	Layout LayoutStyle
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
	s := openScreen()
	if s.cleanup != nil {
		defer s.cleanup()
	}

	m := NewModel(rows, status, opts)
	m.dark = s.dark()

	p := tea.NewProgram(m, s.opts...)
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

// screen is where the Picker draws: the files it reads from and writes to,
// and the Bubble Tea options pointing the program at them.
type screen struct {
	in, out *os.File
	opts    []tea.ProgramOption

	// cleanup closes the files, when they are ours to close. It is nil for
	// the stderr fallback.
	cleanup func()
}

// openScreen opens /dev/tty for the Picker to draw on, falling back to
// stderr when no TTY can be opened. stdout is never used, since a caller
// may pipe it (the Wrapper reads the chosen path from Run's return value,
// not from the program's own output).
func openScreen() screen {
	in, out, err := tea.OpenTTY()
	if err != nil {
		return screen{
			in:   os.Stdin,
			out:  os.Stderr,
			opts: []tea.ProgramOption{tea.WithOutput(os.Stderr)},
		}
	}
	return screen{
		in:   in,
		out:  out,
		opts: []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(out)},
		cleanup: func() {
			_ = in.Close()
			_ = out.Close()
		},
	}
}

// dark reports whether the terminal has a dark background, asked and
// answered before the program starts so the very first frame is already in
// the right palette. Bubble Tea reports the background asynchronously, a
// frame or two in, which repainted the whole Picker on launch. lipgloss
// sends a device-attributes query alongside, so a terminal that ignores the
// background query still ends this one promptly, leaving the palette dark.
func (s screen) dark() bool {
	return lipgloss.HasDarkBackground(s.in, s.out)
}
