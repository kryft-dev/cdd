package picker_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/picker"
)

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
