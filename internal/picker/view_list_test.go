package picker_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/picker"
)

// listModel builds a Model in the flat list layout, sized to width x
// height, over rows.
func listModel(rows []picker.Row, width, height int) picker.Model {
	return sizedModel(rows, picker.LayoutList, width, height)
}

// selBarGlyph is the bar the list layout draws on the selected row.
const selBarGlyph = "▌"

// TestOptions_LayoutDefaultsToGrouped pins the zero Options to the grouped
// layout, so a caller that says nothing keeps the accepted look.
func TestOptions_LayoutDefaultsToGrouped(t *testing.T) {
	m := sizedModel(rowsForKindOrder(), "", 120, 24)
	lines := strings.Split(plain(m.View().Content), "\n")
	if !strings.Contains(lines[0], "type to filter") {
		t.Errorf("first line = %q, want the grouped layout's filter line on top", lines[0])
	}
}

// TestModel_ListLayout_FlatHistoryOrder verifies that the list layout walks
// rows in the order they were given (History order, never-visited last)
// rather than regrouping them under their Kinds.
func TestModel_ListLayout_FlatHistoryOrder(t *testing.T) {
	m := picker.NewModel(rowsForKindOrder(), fakeStatus, picker.Options{Layout: picker.LayoutList})

	want := []string{
		"/root/tools/cdd",
		"/root/work/api",
		"/root/oss/lib",
		"/root/tools/dotfiles",
	}
	for i, wantPath := range want {
		got := chosenAt(t, m, i)
		if got.Project.Path != wantPath {
			t.Errorf("row %d = %q, want %q", i, got.Project.Path, wantPath)
		}
	}
}

// TestModel_ListLayout_StartsWithARowNotAHeader verifies that the frame
// opens on the first Project row: no filter line above it and no Kind
// header taking a line.
func TestModel_ListLayout_StartsWithARowNotAHeader(t *testing.T) {
	m := listModel(rowsForKindOrder(), 120, 24)
	lines := strings.Split(plain(m.View().Content), "\n")

	if !strings.Contains(lines[0], "tools/cdd") {
		t.Errorf("first line = %q, want it to be the tools/cdd row", lines[0])
	}
	if strings.Contains(lines[0], "type to filter") {
		t.Errorf("first line = %q, want the filter prompt below the list, not above it", lines[0])
	}
}

// TestModel_ListLayout_KindPrefixAndPreview verifies a row shows "kind/"
// before the Project name and that the shared preview pane is still drawn.
func TestModel_ListLayout_KindPrefixAndPreview(t *testing.T) {
	m := listModel(rowsForKindOrder(), 120, 24)
	out := plain(m.View().Content)

	if !strings.Contains(out, "work/api") {
		t.Errorf("View() output is missing the %q row:\n%s", "work/api", out)
	}
	if !strings.ContainsAny(out, "╭╮╰╯") {
		t.Errorf("View() output is missing the preview box:\n%s", out)
	}
}

// TestModel_ListLayout_SelectedRowCarriesBar verifies the selected row is
// marked with the "▌" bar, and only the selected row.
func TestModel_ListLayout_SelectedRowCarriesBar(t *testing.T) {
	m := listModel(rowsForKindOrder(), 120, 24)

	if n := strings.Count(plain(m.View().Content), selBarGlyph); n != 1 {
		t.Errorf("View() drew %d %q bars, want exactly 1 (the selected row)", n, selBarGlyph)
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(picker.Model)
	for _, l := range strings.Split(plain(m.View().Content), "\n") {
		if strings.Contains(l, selBarGlyph) && !strings.Contains(l, "work/api") {
			t.Errorf("after one move down the bar is on %q, want it on the work/api row", l)
		}
	}
}

// TestModel_ListLayout_LongKindKeepsTheName verifies that a Kind too long
// for the name column is the part that gives way: the Project name stays
// on screen, and it keeps its own styling rather than being muted along
// with the Kind.
func TestModel_ListLayout_LongKindKeepsTheName(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{
			Kind: "infrastructure-platform",
			Name: "obs",
			Path: "/root/infrastructure-platform/obs",
		}},
	}
	m := listModel(rows, 34, 12) // narrow enough that "kind/" alone overruns

	lines := strings.Split(m.View().Content, "\n")
	row := lines[0]
	if !strings.Contains(plain(row), "obs") {
		t.Fatalf("row = %q, want the Project name %q still drawn", plain(row), "obs")
	}

	// The Kind is muted and the selected row's name is not: the two
	// segments must not share one styling run.
	name := styleOf(row, "obs")
	kind := styleOf(row, "infra")
	if name == "" {
		t.Fatalf("row = %q, could not find a styling run around the name", row)
	}
	if name == kind {
		t.Errorf("name and Kind share the styling run %q, want the Kind muted and the name not", name)
	}
}

// styleOf returns the SGR escape introducing the run of styled text that
// contains want, or "" when want is not found in a styled run.
func styleOf(line, want string) string {
	for _, run := range strings.Split(line, "\x1b[m") {
		if i := strings.LastIndex(run, "m"); i >= 0 && strings.Contains(run[i:], want) {
			if j := strings.Index(run, "\x1b["); j >= 0 {
				return run[j : i+1]
			}
		}
	}
	return ""
}

// TestModel_ListLayout_ScrollsCursorIntoView verifies a list far taller
// than the terminal scrolls so the cursor's row stays drawn.
func TestModel_ListLayout_ScrollsCursorIntoView(t *testing.T) {
	rows := manyRows(40)
	m := listModel(rows, 120, 20)

	for i := 0; i < len(rows)-1; i++ {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(picker.Model)
	}
	last := rows[len(rows)-1].Project.Name
	if !strings.Contains(plain(m.View().Content), last) {
		t.Errorf("after moving the cursor to the last row, View() output does not contain %q", last)
	}
}

// TestModel_ListLayout_NoMatches shows the shared empty-filter message
// rather than an empty pane.
func TestModel_ListLayout_NoMatches(t *testing.T) {
	m := listModel(rowsForKindOrder(), 120, 24)
	for _, r := range "zzzzzz" {
		next, _ := m.Update(tea.KeyPressMsg{Text: string(r)})
		m = next.(picker.Model)
	}
	out := plain(m.View().Content)
	if !strings.Contains(out, "no projects match") {
		t.Errorf("View() with no matches is missing the empty message:\n%s", out)
	}
	if !strings.Contains(out, "nothing selected") {
		t.Errorf("View() with no matches is missing the empty preview:\n%s", out)
	}
}
