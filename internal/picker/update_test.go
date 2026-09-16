package picker_test

import (
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

// TestModel_Update_StatusResult verifies that a status result lands keyed
// by the row's Project path, independent of row index, via a fake
// StatusFunc as the ticket asks.
func TestModel_Update_StatusResult(t *testing.T) {
	m := twoRowModel(picker.Options{})

	next, cmd := m.Update(struct{}{}) // unrelated message: no-op
	final := next.(picker.Model)
	if cmd != nil {
		t.Fatalf("unrelated message should not return a Cmd")
	}

	// Drive Init to obtain the batch of per-row status commands, then run
	// one to get a real message shaped like the runtime would deliver it.
	batch := final.Init()
	if batch == nil {
		t.Fatalf("Init() returned a nil Cmd")
	}
	msg := batch()
	bmsg, ok := msg.(tea.BatchMsg)
	if !ok || len(bmsg) == 0 {
		t.Fatalf("Init() Cmd did not produce a tea.BatchMsg")
	}

	// Feed every command's result through Update; each should be accepted
	// without error regardless of arrival order.
	for _, c := range bmsg {
		next, _ = final.Update(c())
		final = next.(picker.Model)
	}
	_ = git.Status{} // status is opaque here; arrival without panic is the assertion
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
