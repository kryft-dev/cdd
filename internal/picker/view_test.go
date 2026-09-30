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

// allLayouts is every layout the Picker can draw, for the behaviour each
// must share: the frame's height, the alternate screen, equal-width rows
// and windowing the body to the terminal.
var allLayouts = []picker.LayoutStyle{picker.LayoutList}

// sizedModel builds a Model in the given layout, sends it one
// tea.WindowSizeMsg and reports the terminal's background as dark: a sized
// terminal with a settled palette is the state every View test starts
// from, since nothing is drawn before the palette settles.
func sizedModel(rows []picker.Row, layout picker.LayoutStyle, width, height int) picker.Model {
	m := picker.NewModel(rows, noopStatus, picker.Options{Layout: layout})
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	return next.(picker.Model)
}

// ansi matches the SGR escape sequences lipgloss wraps each styled
// segment in. A row's text is split across several of them, so assertions
// about what a line reads strip them first.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain is s with its styling escapes removed, as the terminal shows it.
func plain(s string) string { return ansi.ReplaceAllString(s, "") }

// manyRows builds n rows all in the same directory, so the list body is long
// enough to need scrolling at a modest terminal height.
func manyRows(n int) []picker.Row {
	rows := make([]picker.Row, n)
	for i := range rows {
		name := "project-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		rows[i] = picker.Row{Project: picker.Project{
			Dir:  "~/work/",
			Name: name,
			Path: "/root/work/" + name,
		}}
	}
	return rows
}

// TestModel_View_ListWindowedToHeight verifies that a list far taller than
// the terminal is clipped to Layout.ListHeight rows rather than pushing the
// footer off screen, and that scrolling the cursor to the last row keeps it
// on screen.
func TestModel_View_ListWindowedToHeight(t *testing.T) {
	rows := manyRows(40)
	m := sizedModel(rows, picker.LayoutList, 120, 20)

	lay := picker.ComputeLayout(10, 1, make([]time.Time, len(rows)), time.Now(), 120, 20)

	out := m.View().Content
	lines := strings.Split(out, "\n")

	// Without windowing, every one of the 40 rows would be drawn, pushing the total well past Height; with windowing
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
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(picker.Model)
	}
	out = m.View().Content
	last := rows[len(rows)-1].Project.Name
	if !strings.Contains(out, last) {
		t.Errorf("after moving the cursor to the last row, View() output does not contain %q (row scrolled out of the window)", last)
	}
}

// TestModel_View_RowsShareEqualWidth verifies that every list row renders
// to the same display width, selected or not: the list layout pads every
// row out so the selected row's background spans the pane.
func TestModel_View_RowsShareEqualWidth(t *testing.T) {
	rows := []picker.Row{
		{Project: picker.Project{Dir: "~/work/", Name: "alpha", Path: "/root/work/alpha"}},
		{Project: picker.Project{Dir: "~/work/", Name: "beta", Path: "/root/work/beta"}},
		{Project: picker.Project{Dir: "~/work/", Name: "gamma", Path: "/root/work/gamma"}},
	}
	// Each layout looks for its own row text.
	label := map[picker.LayoutStyle]func(picker.Row) string{
		picker.LayoutList: func(r picker.Row) string { return r.Project.Dir + r.Project.Name },
	}

	for _, layout := range allLayouts {
		t.Run(string(layout), func(t *testing.T) {
			// A narrow terminal keeps the preview pane from being drawn,
			// so each Project name appears exactly once, in its list row.
			m := sizedModel(rows, layout, 30, 40)

			var widths []int
			for _, l := range strings.Split(m.View().Content, "\n") {
				for _, r := range rows {
					if strings.Contains(plain(l), label[layout](r)) {
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
		})
	}
}

// TestModel_View_FrameMatchesTerminalHeight pins the frame to exactly the
// terminal height in every layout, with and without the preview pane. One
// line taller and Bubble Tea's renderer drops a line off the top.
func TestModel_View_FrameMatchesTerminalHeight(t *testing.T) {
	rows := manyRows(11)
	for _, layout := range allLayouts {
		t.Run(string(layout), func(t *testing.T) {
			for _, width := range []int{110, 45} {
				for _, height := range []int{40, 30, 24, 14, 9} {
					m := sizedModel(rows, layout, width, height)
					lines := strings.Split(m.View().Content, "\n")
					if len(lines) != height {
						t.Errorf("View() at %dx%d produced %d lines, want exactly %d", width, height, len(lines), height)
					}
				}
			}
		})
	}
}

// TestModel_View_FilterLineSitsWhereTheLayoutPutsIt pins the filter line
// to where each layout puts it: directly below the list body in the list
// layout, where fzf users expect the prompt.
func TestModel_View_FilterLineSitsWhereTheLayoutPutsIt(t *testing.T) {
	rows := manyRows(11)
	const width, height = 110, 24
	lay := picker.ComputeLayout(20, 1, make([]time.Time, len(rows)), time.Now(), width, height)

	promptLine := map[picker.LayoutStyle]int{
		picker.LayoutList: lay.ListHeight,
	}
	for _, layout := range allLayouts {
		t.Run(string(layout), func(t *testing.T) {
			m := sizedModel(rows, layout, width, height)
			lines := strings.Split(plain(m.View().Content), "\n")
			want := promptLine[layout]
			if !strings.Contains(lines[want], "type to filter") {
				t.Errorf("line %d = %q, want the filter prompt there", want, lines[want])
			}
			for i, l := range lines {
				if i != want && strings.Contains(l, "type to filter") {
					t.Errorf("line %d = %q also holds the filter prompt, want it only on line %d", i, l, want)
				}
			}
		})
	}
}

// TestModel_View_UsesAlternateScreen pins every layout to the alternate
// screen so nothing is left above the shell prompt after a Jump or cancel.
func TestModel_View_UsesAlternateScreen(t *testing.T) {
	for _, layout := range allLayouts {
		t.Run(string(layout), func(t *testing.T) {
			if m := sizedModel(manyRows(3), layout, 100, 30); !m.View().AltScreen {
				t.Errorf("View().AltScreen = false, want true")
			}
		})
	}
	empty := picker.NewModel(nil, noopStatus, picker.Options{})
	if !empty.View().AltScreen {
		t.Errorf("empty-History View().AltScreen = false, want true")
	}
}
