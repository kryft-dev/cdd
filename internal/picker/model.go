package picker

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/match"
)

// focus tracks which part of the Picker receives key presses. Only the vim
// key map ever leaves the list unfocused (it starts on the list itself, but
// f or / can move focus to the filter).
type focus int

const (
	focusList focus = iota
	focusFilter
)

// hit is one row along with where, if anywhere, the current query matched
// its Project's shown path (Dir then Name), for highlighting.
type hit struct {
	row     Row
	matches []int // rune indexes into the matched string, for highlighting
}

// Model is the Picker's Bubble Tea Model. It never mutates rows: filtering
// and grouping are recomputed from it as the query and window size change.
type Model struct {
	rows   []Row
	status StatusFunc
	vim    bool
	layout Layout

	actions map[string]action.Action // keyed by Action.Key
	hints   []action.Action          // the bound Actions in hint order
	noHints bool                     // Options.HideHints
	help    bool                     // the vim help overlay is open
	runner  action.Runner
	copy    func(text string) (bool, error)

	query  string
	focus  focus
	cursor int // index into the current visible (filtered) rows

	statuses map[string]git.Status // keyed by Project.Path

	width, height int
	dark          bool

	// paletteSettled reports whether the terminal has had its say about
	// its background colour, one way or the other. Nothing is drawn until
	// it has.
	paletteSettled bool

	// message is the one line the footer shows in place of the key hints
	// until the next key press: a detached Action's failure to start, or
	// that a copy succeeded (messageOK, shown as good news, not an error).
	message   string
	messageOK bool

	chosen       bool
	chosenRow    Row
	chosenAction *action.Action
	quitting     bool
}

// NewModel builds the Picker's initial Model from rows already in History
// order.
func NewModel(rows []Row, status StatusFunc, opts Options) Model {
	f := focusFilter
	if opts.Vim {
		f = focusList
	}
	layout := opts.Layout
	if layout == "" {
		layout = LayoutList
	}
	actions := make(map[string]action.Action, len(opts.Actions))
	var hints []action.Action
	for _, a := range opts.Actions {
		if a.Key == "" {
			continue
		}
		actions[a.Key] = a
		if a.Key == "enter" {
			// Whatever holds enter heads the hints, as the way in.
			hints = append([]action.Action{a}, hints...)
		} else {
			hints = append(hints, a)
		}
	}
	runner := opts.Runner
	if runner == nil {
		runner = action.ExecRunner{}
	}
	copyFn := opts.Copy
	if copyFn == nil {
		copyFn = action.Copy
	}
	return Model{
		actions:  actions,
		hints:    hints,
		noHints:  opts.HideHints,
		runner:   runner,
		copy:     copyFn,
		rows:     rows,
		status:   status,
		vim:      opts.Vim,
		layout:   layout,
		query:    opts.Query,
		focus:    f,
		statuses: make(map[string]git.Status, len(rows)),

		// Dark until the terminal says otherwise, as lipgloss itself
		// assumes: Run settles it before the first frame, and a light
		// terminal that answers neither query is rarer than a dark one.
		dark: true,
	}
}

// Chosen returns the Row an Action key has chosen, and whether one has
// been chosen yet. It lets a caller (or a test) read the outcome without
// waiting for the Bubble Tea runtime to hand back the final Model.
func (m Model) Chosen() (Row, bool) {
	return m.chosenRow, m.chosen
}

// ChosenAction returns the Action that chose the Row, or nil when none has.
func (m Model) ChosenAction() *action.Action {
	return m.chosenAction
}

// statusResultMsg is the result of one StatusFunc call, keyed by the
// Project's path rather than its row index: the fuzzy filter reorders
// visible rows while calls are still in flight.
type statusResultMsg struct {
	path   string
	status git.Status
}

// paletteDeadline is how long the first frame waits on the terminal's
// background colour before being drawn in the dark palette anyway. A
// terminal answers in a few milliseconds; one that never answers must not
// hold the Picker off the screen.
const paletteDeadline = 50 * time.Millisecond

// paletteDeadlineMsg says the terminal has had long enough to report its
// background colour.
type paletteDeadlineMsg struct{}

// Init fires one command per row that fetches its git status, fanned out
// through tea.Batch and bounded by a semaphore so a large History does not
// spawn unbounded concurrent git processes. It also requests the terminal
// background colour, which the first frame waits on, and starts the
// deadline that wait is given.
func (m Model) Init() tea.Cmd {
	sem := make(chan struct{}, concurrency)
	cmds := make([]tea.Cmd, 0, len(m.rows)+2)
	cmds = append(cmds, tea.RequestBackgroundColor, tea.Tick(paletteDeadline, func(time.Time) tea.Msg {
		return paletteDeadlineMsg{}
	}))
	for _, r := range m.rows {
		cmds = append(cmds, statusCmd(m.status, r.Project.Path, sem))
	}
	return tea.Batch(cmds...)
}

// statusCmd builds the tea.Cmd for one row's status check. The outer
// function captures the path; only the inner func() tea.Msg runs on its own
// goroutine, where the semaphore is acquired and released.
func statusCmd(status StatusFunc, path string, sem chan struct{}) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		return statusResultMsg{path: path, status: status(context.Background(), path)}
	}
}

// visibleMatches returns the current query's matches over rows: every row
// in History order when the query is empty, otherwise ranked by the match
// package, equally good matches keeping History order.
func (m Model) visibleMatches() []hit {
	q := match.Parse(m.query)
	if q.Empty() {
		out := make([]hit, len(m.rows))
		for i, r := range m.rows {
			out[i] = hit{row: r}
		}
		return out
	}

	var out []hit
	var results []match.Result
	for _, r := range m.rows {
		res, ok := q.Match(r.Project.Dir, r.Project.Name)
		if !ok {
			continue
		}
		out = append(out, hit{row: r, matches: res.Indexes})
		results = append(results, res)
	}
	sort.Stable(byResult{out, results})
	return out
}

// byResult sorts hits by their match.Result, keeping hits and results in
// step.
type byResult struct {
	hits    []hit
	results []match.Result
}

func (b byResult) Len() int           { return len(b.hits) }
func (b byResult) Less(i, j int) bool { return match.Less(b.results[i], b.results[j]) }
func (b byResult) Swap(i, j int) {
	b.hits[i], b.hits[j] = b.hits[j], b.hits[i]
	b.results[i], b.results[j] = b.results[j], b.results[i]
}

// visibleRows returns the current query's matches in the order the layout
// draws them. It is the order the cursor indexes into.
func (m Model) visibleRows() []hit {
	return m.visibleMatches()
}
