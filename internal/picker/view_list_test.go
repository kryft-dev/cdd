package picker_test

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/picker"
)

// listModel builds a Model in the flat list layout, sized to width x
// height, over rows.
func listModel(rows []picker.Row, width, height int) picker.Model {
	return sizedModel(rows, picker.LayoutList, width, height)
}

// selBarGlyph is the bar the list layout draws on the selected row.
const selBarGlyph = "▌"

// TestOptions_LayoutDefaultsToList pins the zero Options to the list
// layout, so a caller that says nothing gets the only layout there is.
func TestOptions_LayoutDefaultsToList(t *testing.T) {
	m := sizedModel(historyRows(), "", 120, 24)
	lines := strings.Split(plain(m.View().Content), "\n")
	if !strings.Contains(lines[0], "tools/cdd") {
		t.Errorf("first line = %q, want the list layout's first row on top", lines[0])
	}
}

// TestModel_ListLayout_FlatHistoryOrder verifies that the list layout walks
// rows in the order they were given (History order) rather than regrouping
// them by parent directory.
func TestModel_ListLayout_FlatHistoryOrder(t *testing.T) {
	m := picker.NewModel(historyRows(), fakeStatus, picker.Options{Layout: picker.LayoutList, Actions: action.Builtins()})

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
// opens on the first Project row: no filter line above it and no header
// taking a line.
func TestModel_ListLayout_StartsWithARowNotAHeader(t *testing.T) {
	m := listModel(historyRows(), 120, 24)
	lines := strings.Split(plain(m.View().Content), "\n")

	if !strings.Contains(lines[0], "tools/cdd") {
		t.Errorf("first line = %q, want it to be the tools/cdd row", lines[0])
	}
	if strings.Contains(lines[0], "type to filter") {
		t.Errorf("first line = %q, want the filter prompt below the list, not above it", lines[0])
	}
}

// TestModel_ListLayout_DirPrefixAndPreview verifies a row shows its parent
// directory before the Project name and that the shared preview pane is
// still drawn.
func TestModel_ListLayout_DirPrefixAndPreview(t *testing.T) {
	m := listModel(historyRows(), 120, 24)
	out := plain(m.View().Content)

	if !strings.Contains(out, "~/work/api") {
		t.Errorf("View() output is missing the %q row:\n%s", "~/work/api", out)
	}
	if !strings.ContainsAny(out, "╭╮╰╯") {
		t.Errorf("View() output is missing the preview box:\n%s", out)
	}
}

// TestModel_ListLayout_SelectedRowCarriesBar verifies the selected row is
// marked with the "▌" bar, and only the selected row.
func TestModel_ListLayout_SelectedRowCarriesBar(t *testing.T) {
	m := listModel(historyRows(), 120, 24)

	if n := strings.Count(plain(m.View().Content), selBarGlyph); n != 1 {
		t.Errorf("View() drew %d %q bars, want exactly 1 (the selected row)", n, selBarGlyph)
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(picker.Model)
	for _, l := range strings.Split(plain(m.View().Content), "\n") {
		if strings.Contains(l, selBarGlyph) && !strings.Contains(l, "~/work/api") {
			t.Errorf("after one move down the bar is on %q, want it on the work/api row", l)
		}
	}
}

// TestModel_ListLayout_LongDirKeepsTheName verifies that a parent directory
// too long for the name column is the part that gives way, from its start:
// the Project name and the directory's end stay on screen, and the name
// keeps its own styling rather than being muted along with the directory.
func TestModel_ListLayout_LongDirKeepsTheName(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{
			Dir:  "~/work/infrastructure-platform/",
			Name: "obs",
			Path: "/root/work/infrastructure-platform/obs",
		}},
	}
	m := listModel(rows, 34, 12) // narrow enough that the directory alone overruns

	lines := strings.Split(m.View().Content, "\n")
	row := lines[0]
	if !strings.Contains(plain(row), "obs") {
		t.Fatalf("row = %q, want the Project name %q still drawn", plain(row), "obs")
	}
	if !strings.Contains(plain(row), "…") || !strings.Contains(plain(row), "platform/") || strings.Contains(plain(row), "~/work") {
		t.Errorf("row = %q, want the directory's start dropped behind \"…\" and its end kept", plain(row))
	}

	// The directory is muted and the selected row's name is not: the two
	// segments must not share one styling run.
	name := styleOf(row, "obs")
	dir := styleOf(row, "platform")
	if name == "" {
		t.Fatalf("row = %q, could not find a styling run around the name", row)
	}
	if name == dir {
		t.Errorf("name and directory share the styling run %q, want the directory muted and the name not", name)
	}
}

// TestModel_ListLayout_FilterMatchesDirAndHighlights verifies the filter
// matches the parent directory as well as the name, and that the matched
// runes of a truncated directory still land on the right characters.
func TestModel_ListLayout_FilterMatchesDirAndHighlights(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{Dir: "~/work/infrastructure-platform/", Name: "obs", Path: "/w/i/obs"}},
		{Project: picker.Project{Dir: "~/tools/", Name: "cdd", Path: "/t/cdd"}},
	}
	m := listModel(rows, 34, 12)
	for _, r := range "platf" {
		next, _ := m.Update(tea.KeyPressMsg{Text: string(r)})
		m = next.(picker.Model)
	}

	row := strings.Split(m.View().Content, "\n")[0]
	if !strings.Contains(plain(row), "obs") {
		t.Fatalf("row = %q, want the obs row, matched by its directory", plain(row))
	}
	var highlighted strings.Builder
	for _, sm := range underlined.FindAllStringSubmatch(row, -1) {
		highlighted.WriteString(sm[1])
	}
	if got := highlighted.String(); got != "platf" {
		t.Errorf("highlighted runes = %q, want %q", got, "platf")
	}
}

// underlined matches one rune drawn in the underlined match style.
var underlined = regexp.MustCompile(`\x1b\[1;4;[0-9;]*m([^\x1b])\x1b\[m`)

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
	m := listModel(historyRows(), 120, 24)
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
