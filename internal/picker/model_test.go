package picker_test

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/picker"
)

// fakeStatus is a StatusFunc that reports a fixed clean-repo status for
// every row, for tests that only care about ordering or filtering.
func fakeStatus(context.Context, string) git.Status {
	return git.Status{Kind: git.Found}
}

// noopStatus never resolves anything useful; it is used when a test does
// not care what status lands.
func noopStatus(context.Context, string) git.Status {
	return git.Status{Kind: git.NotRepo}
}

func rowsForKindOrder() []picker.Row {
	// History order: "work" holds the most recent Visit, so it must lead
	// the Kind groups even though "tools" and "oss" rows also exist.
	return []picker.Row{
		{Project: picker.Project{Kind: "tools", Name: "cdd", Path: "/root/tools/cdd"}, LastVisit: time.Unix(3000, 0)},
		{Project: picker.Project{Kind: "work", Name: "api", Path: "/root/work/api"}, LastVisit: time.Unix(5000, 0)},
		{Project: picker.Project{Kind: "oss", Name: "lib", Path: "/root/oss/lib"}, LastVisit: time.Unix(1000, 0)},
		{Project: picker.Project{Kind: "tools", Name: "dotfiles", Path: "/root/tools/dotfiles"}, LastVisit: time.Unix(2000, 0)},
	}
}

// TestModel_KindOrder verifies that moving the cursor through the grouped
// list visits Kinds in the order they first appear among rows (History
// order), not alphabetically, and that each Kind's own rows stay in the
// order they were given.
func TestModel_KindOrder(t *testing.T) {
	rows := rowsForKindOrder()
	m := picker.NewModel(rows, fakeStatus, picker.Options{})

	// Rows come in as: tools/cdd, work/api, oss/lib, tools/dotfiles.
	// Grouped by first appearance: tools{cdd, dotfiles}, work{api}, oss{lib}.
	want := []string{
		"/root/tools/cdd",
		"/root/tools/dotfiles",
		"/root/work/api",
		"/root/oss/lib",
	}

	for i, wantPath := range want {
		got := chosenAt(t, m, i)
		if got.Project.Path != wantPath {
			t.Errorf("row %d = %q, want %q", i, got.Project.Path, wantPath)
		}
	}
}

// chosenAt moves the cursor to index i from the top and returns the Row
// enter would choose there, using the default key map.
func chosenAt(t *testing.T, m picker.Model, i int) picker.Row {
	t.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(picker.Model)
	for n := 0; n < i; n++ {
		next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(picker.Model)
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	final := next.(picker.Model)
	row, ok := final.Chosen()
	if !ok {
		t.Fatalf("row %d: enter did not choose a Row", i)
	}
	return row
}

// TestModel_FuzzyFilter verifies that typing a query narrows the visible
// rows to those whose Project path fuzzy-matches it.
func TestModel_FuzzyFilter(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{Kind: "work", Name: "api-gateway", Path: "/root/work/api-gateway"}},
		{Project: picker.Project{Kind: "work", Name: "billing", Path: "/root/work/billing"}},
		{Project: picker.Project{Kind: "oss", Name: "bubbletea", Path: "/root/oss/bubbletea"}},
	}
	m := picker.NewModel(rows, noopStatus, picker.Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(picker.Model)

	for _, r := range []rune("billing") {
		next, _ = m.Update(tea.KeyPressMsg{Text: string(r)})
		m = next.(picker.Model)
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	final := next.(picker.Model)
	row, ok := final.Chosen()
	if !ok {
		t.Fatalf("enter did not choose a Row after filtering")
	}
	if row.Project.Name != "billing" {
		t.Errorf("chosen Project = %q, want %q", row.Project.Name, "billing")
	}
}
