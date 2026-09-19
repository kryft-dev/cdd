package picker_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/picker"
)

// ansi matches the SGR escape sequences lipgloss wraps each styled
// segment in. A row's text is split across several of them, so assertions
// about what a line reads strip them first.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain is s with its styling escapes removed, as the terminal shows it.
func plain(s string) string { return ansi.ReplaceAllString(s, "") }

// listModel builds a Model in the flat list layout, sized to width x
// height, over rows.
func listModel(rows []picker.Row, width, height int) picker.Model {
	m := picker.NewModel(rows, noopStatus, picker.Options{Layout: picker.LayoutList})
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return next.(picker.Model)
}

// selBarGlyph is the bar the list layout draws on the selected row.
const selBarGlyph = "▌"

// TestOptions_LayoutDefaultsToGrouped pins the zero Options to the grouped
// layout, so a caller that says nothing keeps the accepted look.
func TestOptions_LayoutDefaultsToGrouped(t *testing.T) {
	m := listModelWithLayout(t, "")
	lines := strings.Split(plain(m.View().Content), "\n")
	if !strings.Contains(lines[0], "type to filter") {
		t.Errorf("first line = %q, want the grouped layout's filter line on top", lines[0])
	}
}

// listModelWithLayout builds a sized Model over rowsForKindOrder under the
// named layout.
func listModelWithLayout(t *testing.T, layout picker.LayoutStyle) picker.Model {
	t.Helper()
	m := picker.NewModel(rowsForKindOrder(), noopStatus, picker.Options{Layout: layout})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	return next.(picker.Model)
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

// TestModel_ListLayout_PromptBelowList pins the filter prompt to the line
// directly below the list body, where fzf users expect it, with the footer
// underneath.
func TestModel_ListLayout_PromptBelowList(t *testing.T) {
	rows := rowsForKindOrder()
	m := listModel(rows, 120, 24)
	lay := picker.ComputeLayout(20, 1, make([]time.Time, len(rows)), time.Now(), 120, 24)

	lines := strings.Split(plain(m.View().Content), "\n")
	prompt := lines[lay.ListHeight]
	if !strings.Contains(prompt, "type to filter") {
		t.Errorf("line %d = %q, want the filter prompt directly below the list", lay.ListHeight, prompt)
	}

	sawKeys := false
	for _, l := range lines[lay.ListHeight+1:] {
		if strings.Contains(l, "move") && strings.Contains(l, "jump") {
			sawKeys = true
		}
	}
	if !sawKeys {
		t.Errorf("footer keys line is missing below the prompt:\n%s", m.View().Content)
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

// TestModel_ListLayout_RowsShareEqualWidth verifies every list row renders
// to the same display width, so the selected row's background highlight
// spans the pane instead of stopping at the text.
func TestModel_ListLayout_RowsShareEqualWidth(t *testing.T) {
	rows := rowsForKindOrder()
	m := listModel(rows, 40, 24) // narrow: no preview pane, names appear once

	var widths []int
	for _, l := range strings.Split(m.View().Content, "\n") {
		for _, r := range rows {
			if strings.Contains(plain(l), r.Project.Kind+"/"+r.Project.Name) {
				widths = append(widths, lipgloss.Width(l))
			}
		}
	}
	if len(widths) != len(rows) {
		t.Fatalf("found %d row lines, want %d", len(widths), len(rows))
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Errorf("row %d width = %d, want %d (same as row 0, selected or not)", i, widths[i], widths[0])
		}
	}
}

// TestModel_ListLayout_FrameMatchesTerminalHeight pins the list layout's
// frame to exactly the terminal height, at every size the grouped layout
// is pinned at.
func TestModel_ListLayout_FrameMatchesTerminalHeight(t *testing.T) {
	rows := manyRows(11)
	for _, width := range []int{110, 45} {
		for _, height := range []int{40, 30, 24, 14, 9} {
			m := listModel(rows, width, height)
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != height {
				t.Errorf("View() at %dx%d produced %d lines, want exactly %d", width, height, len(lines), height)
			}
		}
	}
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

// TestModel_ListLayout_UsesAlternateScreen pins the list layout to the
// alternate screen, like the grouped one.
func TestModel_ListLayout_UsesAlternateScreen(t *testing.T) {
	m := listModel(manyRows(3), 100, 30)
	if !m.View().AltScreen {
		t.Errorf("View().AltScreen = false, want true")
	}
}
