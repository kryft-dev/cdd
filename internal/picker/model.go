package picker

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/kryft-dev/cdd/internal/git"
)

// focus tracks which part of the Picker receives key presses. Only the vim
// key map ever leaves the list unfocused (it starts on the list itself, but
// f or / can move focus to the filter).
type focus int

const (
	focusList focus = iota
	focusFilter
)

// match is one row along with where, if anywhere, the current query matched
// its Project's path, for highlighting and ordering within its Kind group.
type match struct {
	row     Row
	matches []int // rune indexes into the matched string, for highlighting
}

// Model is the Picker's Bubble Tea Model. It never mutates rows: filtering
// and grouping are recomputed from it as the query and window size change.
type Model struct {
	rows   []Row
	status StatusFunc
	vim    bool
	layout LayoutStyle

	query  string
	focus  focus
	cursor int // index into the current visible/filtered+grouped rows

	statuses map[string]git.Status // keyed by Project.Path

	width, height int
	dark          bool

	// paletteSettled reports whether the terminal has had its say about
	// its background colour, one way or the other. Nothing is drawn until
	// it has.
	paletteSettled bool

	chosen    bool
	chosenRow Row
	quitting  bool
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
		layout = LayoutGrouped
	}
	return Model{
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

// Chosen returns the Row an "enter" press has chosen, and whether one has
// been chosen yet. It lets a caller (or a test) read the outcome without
// waiting for the Bubble Tea runtime to hand back the final Model.
func (m Model) Chosen() (Row, bool) {
	return m.chosenRow, m.chosen
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

// visible returns the current query's matches over rows, in row order
// (which preserves History order within a match set, since fuzzy.Find is
// stable relative to its input order for equal scores is not guaranteed,
// but grouping below only depends on first-appearance order of Kind, not on
// score order).
func (m Model) visibleMatches() []match {
	if m.query == "" {
		out := make([]match, len(m.rows))
		for i, r := range m.rows {
			out[i] = match{row: r}
		}
		return out
	}

	paths := make([]string, len(m.rows))
	for i, r := range m.rows {
		paths[i] = r.Project.Path
	}
	results := fuzzy.Find(m.query, paths)

	out := make([]match, len(results))
	for i, res := range results {
		out[i] = match{row: m.rows[res.Index], matches: res.MatchedIndexes}
	}
	return out
}

// kindGroup is one Kind's header plus the matches that fall under it, in
// the order Kinds first appear among the visible matches.
type kindGroup struct {
	kind    string
	matches []match
}

// groupByKind groups matches under their Project's Kind, ordering Kinds by
// first appearance in matches (which is History order, or fuzzy-ranked
// order when a query is active) and keeping each Kind's own rows in that
// same order.
func groupByKind(matches []match) []kindGroup {
	var groups []kindGroup
	index := make(map[string]int)
	for _, mt := range matches {
		kind := mt.row.Project.Kind
		i, ok := index[kind]
		if !ok {
			i = len(groups)
			index[kind] = i
			groups = append(groups, kindGroup{kind: kind})
		}
		groups[i].matches = append(groups[i].matches, mt)
	}
	return groups
}

// visibleGroups returns the current query's matches grouped by Kind, in the
// order rows are actually displayed: the order the Picker's cursor and
// filtering operate on.
func (m Model) visibleGroups() []kindGroup {
	return groupByKind(m.visibleMatches())
}

// visibleRows returns the current query's matches in the order the active
// layout draws them: grouped by Kind for LayoutGrouped, flat History order
// for LayoutList. It is the order the cursor indexes into.
func (m Model) visibleRows() []match {
	matches := m.visibleMatches()
	if m.layout == LayoutList {
		return matches
	}
	return flatten(groupByKind(matches))
}

// flatten lays a Kind grouping out as a single ordered slice of matches,
// matching the row order the list draws (header lines aside).
func flatten(groups []kindGroup) []match {
	var out []match
	for _, g := range groups {
		out = append(out, g.matches...)
	}
	return out
}
