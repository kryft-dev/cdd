package picker_test

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/picker"
)

// manyRows builds n rows all under the same Kind, so the list body is long
// enough to need scrolling at a modest terminal height.
func manyRows(n int) []picker.Row {
	rows := make([]picker.Row, n)
	for i := range rows {
		name := "project-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		rows[i] = picker.Row{Project: picker.Project{
			Kind: "work",
			Name: name,
			Path: "/root/work/" + name,
		}}
	}
	return rows
}

// TestModel_View_PreviewUnderFilterLine verifies that the preview box does
// not land on the same row as the filter line: the first rendered line
// must be the filter prompt, with no preview box border on it.
func TestModel_View_PreviewUnderFilterLine(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{Kind: "work", Name: "alpha", Path: "/root/work/alpha"}},
	}
	m := picker.NewModel(rows, noopStatus, picker.Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(picker.Model)

	out := m.View().Content
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatalf("View() produced no lines")
	}
	first := lines[0]
	if !strings.Contains(first, "type to filter") {
		t.Errorf("first line = %q, want it to contain the filter prompt", first)
	}
	if strings.ContainsAny(first, "╭╮╰╯│─") {
		t.Errorf("first line = %q, should not contain any preview box border", first)
	}
}

// TestModel_View_ListWindowedToHeight verifies that a list far taller than
// the terminal is clipped to Layout.ListHeight rows rather than pushing the
// footer off screen, and that scrolling the cursor to the last row keeps it
// on screen.
func TestModel_View_ListWindowedToHeight(t *testing.T) {
	rows := manyRows(40)
	m := picker.NewModel(rows, noopStatus, picker.Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	m = next.(picker.Model)

	lay := picker.ComputeLayout(10, 1, make([]time.Time, len(rows)), time.Now(), 120, 20)

	out := m.View().Content
	lines := strings.Split(out, "\n")

	// Without windowing, every one of the 40 rows plus its Kind header
	// would be drawn, pushing the total well past Height; with windowing
	// the whole frame stays close to the terminal height.
	if len(lines) > lay.ListHeight+6 {
		t.Errorf("View() produced %d lines at Height 20 (ListHeight=%d); footer likely pushed off screen:\n%s", len(lines), lay.ListHeight, out)
	}

	sawKeys := false
	for _, l := range lines {
		if strings.Contains(l, "move") && strings.Contains(l, "jump") {
			sawKeys = true
		}
	}
	if !sawKeys {
		t.Errorf("View() output is missing the footer keys line:\n%s", out)
	}

	// Move the cursor to the last row and confirm its name still appears
	// in the rendered output (i.e. it scrolled into the window).
	for i := 0; i < len(rows)-1; i++ {
		next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(picker.Model)
	}
	out = m.View().Content
	last := rows[len(rows)-1].Project.Name
	if !strings.Contains(out, last) {
		t.Errorf("after moving the cursor to the last row, View() output does not contain %q (row scrolled out of the window)", last)
	}
}
