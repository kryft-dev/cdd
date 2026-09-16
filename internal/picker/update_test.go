package picker_test

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/picker"
)

func twoRowModel(opts picker.Options) picker.Model {
	rows := []picker.Row{
		{Project: picker.Project{Kind: "work", Name: "alpha", Path: "/root/work/alpha"}},
		{Project: picker.Project{Kind: "work", Name: "beta", Path: "/root/work/beta"}},
	}
	return picker.NewModel(rows, noopStatus, opts)
}

func TestModel_Update_EscCancels(t *testing.T) {
	m := twoRowModel(picker.Options{})
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	final := next.(picker.Model)

	if _, ok := final.Chosen(); ok {
		t.Fatalf("Chosen() ok = true after esc, want false")
	}
	if cmd == nil {
		t.Fatalf("esc should return tea.Quit, got nil Cmd")
	}
}

func TestModel_Update_CtrlUClearsFilter(t *testing.T) {
	m := twoRowModel(picker.Options{Query: "alpha"})
	next, _ := m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	final := next.(picker.Model)

	next, _ = final.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	chosen := next.(picker.Model)
	row, ok := chosen.Chosen()
	if !ok {
		t.Fatalf("enter after ctrl+u did not choose a Row")
	}
	// With the query cleared, the first row (alpha, History order) is
	// chosen rather than a filtered single match.
	if row.Project.Name != "alpha" {
		t.Errorf("chosen Project = %q, want %q", row.Project.Name, "alpha")
	}
}

// keyedStatus is a StatusFunc that returns a distinct git.Status per path,
// so a test can tell whether a result landed on the row it was meant for.
func keyedStatus(statuses map[string]git.Status) picker.StatusFunc {
	return func(_ context.Context, path string) git.Status {
		return statuses[path]
	}
}

// TestModel_Update_StatusResult verifies that a status result lands keyed
// by the row's Project path, independent of row index, via a fake
// StatusFunc as the ticket asks: each row gets a distinct git.Status, the
// resulting messages are fed through Update in reverse arrival order, and
// the rendered View still shows the right glyph on the right row.
func TestModel_Update_StatusResult(t *testing.T) {
	statuses := map[string]git.Status{
		"/root/work/alpha": {Kind: git.Found, State: git.Dirty},
		"/root/work/beta":  {Kind: git.NotRepo},
	}
	m := picker.NewModel(
		[]picker.Row{
			{Project: picker.Project{Kind: "work", Name: "alpha", Path: "/root/work/alpha"}},
			{Project: picker.Project{Kind: "work", Name: "beta", Path: "/root/work/beta"}},
		},
		keyedStatus(statuses),
		picker.Options{},
	)
	// A narrow terminal keeps the preview pane from being drawn, so each
	// Project path appears exactly once, in its list row.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 40})
	final := next.(picker.Model)

	// Drive Init to obtain the batch of per-row status commands, then run
	// each to get real messages shaped like the runtime would deliver
	// them.
	batch := final.Init()
	if batch == nil {
		t.Fatalf("Init() returned a nil Cmd")
	}
	msg := batch()
	bmsg, ok := msg.(tea.BatchMsg)
	if !ok || len(bmsg) == 0 {
		t.Fatalf("Init() Cmd did not produce a tea.BatchMsg")
	}
	results := make([]tea.Msg, 0, len(bmsg))
	for _, c := range bmsg {
		results = append(results, c())
	}

	// Feed the results through Update in reverse order: if results were
	// keyed by arrival index rather than by Project path, the last row's
	// status (beta, NotRepo) would land on the first row instead.
	for i := len(results) - 1; i >= 0; i-- {
		next, _ = final.Update(results[i])
		final = next.(picker.Model)
	}

	out := final.View().Content
	lines := strings.Split(out, "\n")
	var alphaLine, betaLine string
	for _, l := range lines {
		if strings.Contains(l, "alpha") {
			alphaLine = l
		}
		if strings.Contains(l, "beta") {
			betaLine = l
		}
	}
	if alphaLine == "" || betaLine == "" {
		t.Fatalf("could not find both row lines in View():\n%s", out)
	}
	if !strings.Contains(alphaLine, "●") {
		t.Errorf("alpha row = %q, want it to contain the dirty glyph ●", alphaLine)
	}
	if !strings.Contains(betaLine, "—") {
		t.Errorf("beta row = %q, want it to contain the not-a-repo glyph —", betaLine)
	}
}

func TestModel_Update_VimKeys(t *testing.T) {
	m := twoRowModel(picker.Options{Vim: true})

	next, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = next.(picker.Model)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	final := next.(picker.Model)

	row, ok := final.Chosen()
	if !ok {
		t.Fatalf("enter did not choose a Row in vim mode")
	}
	if row.Project.Name != "beta" {
		t.Errorf("chosen Project = %q, want %q (j should move down)", row.Project.Name, "beta")
	}
}

func TestModel_Update_VimQuitsOnQ(t *testing.T) {
	m := twoRowModel(picker.Options{Vim: true})
	next, cmd := m.Update(tea.KeyPressMsg{Text: "q"})
	final := next.(picker.Model)

	if _, ok := final.Chosen(); ok {
		t.Fatalf("Chosen() ok = true after q, want false")
	}
	if cmd == nil {
		t.Fatalf("q on the list should return tea.Quit, got nil Cmd")
	}
}
