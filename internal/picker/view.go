package picker

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/git"
)

// View renders the current frame: the filter line, the grouped list (with a
// preview pane beside it when there is room), and the footer.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	if len(m.rows) == 0 {
		return tea.NewView(m.emptyHistoryView())
	}

	t := newTheme(m.dark)
	now := time.Now()
	groups := m.visibleGroups()
	rows := flatten(groups)

	longestName := 0
	for _, mt := range rows {
		if n := len([]rune(mt.row.Project.Name)); n > longestName {
			longestName = n
		}
	}
	widestStatus := 1
	times := make([]time.Time, 0, len(rows))
	for _, mt := range rows {
		st, loaded := m.statuses[mt.row.Project.Path]
		if w := statusClusterWidth(st, loaded); w > widestStatus {
			widestStatus = w
		}
		times = append(times, mt.row.LastVisit)
	}

	width, height := m.width, m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	lay := ComputeLayout(longestName, widestStatus, times, now, width, height)

	var b strings.Builder
	b.WriteString(m.filterLine(t))
	b.WriteString("\n")

	list := m.listView(t, groups, rows, lay, now)
	if lay.ShowPreview {
		// The preview column is joined under a matching blank line so its
		// box's top border lands on the list's first row, not on the
		// filter line above.
		right := "\n" + m.previewView(t, rows, lay, now)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, list, " ", right))
	} else {
		b.WriteString(list)
	}

	b.WriteString("\n")
	b.WriteString(m.footerView(t, width, len(rows), lay))

	return tea.NewView(b.String())
}

// emptyHistoryView is shown when there are no rows at all: an empty
// History has nothing for the Picker to list.
func (m Model) emptyHistoryView() string {
	t := newTheme(m.dark)
	return t.muted_().Render("run cdd scan to seed History")
}

// filterLine renders the "❯ " prompt, the query, and a muted placeholder
// when the query is empty.
func (m Model) filterLine(t theme) string {
	prompt := t.fg(t.accent).Bold(true).Render("❯ ")
	if m.query == "" {
		return prompt + t.muted_().Render("type to filter")
	}
	return prompt + m.query
}

// listView renders the grouped list body: a header line per Kind, then its
// rows, with the cursor's row carrying the caret and accent name. The body
// is windowed to exactly Layout.ListHeight lines, scrolled so the cursor's
// line (counting Kind header lines) stays on screen.
func (m Model) listView(t theme, groups []kindGroup, rows []match, lay Layout, now time.Time) string {
	var lines []string
	i := 0
	cursorLine := 0
	for _, g := range groups {
		rule := t.rule_(max(lay.ListWidth-len([]rune(g.kind))-1, 0))
		lines = append(lines, t.accentBold().Render(g.kind)+" "+rule)
		for _, mt := range g.matches {
			if i == m.cursor {
				cursorLine = len(lines)
			}
			lines = append(lines, m.rowView(t, mt, i == m.cursor, lay, now))
			i++
		}
	}
	if len(rows) == 0 {
		lines = append(lines, t.muted_().Render("no projects match"))
	}

	listH := max(lay.ListHeight, 1)
	start := 0
	if cursorLine >= listH {
		start = cursorLine - listH + 1
	}
	windowed := make([]string, listH)
	for i := range windowed {
		if idx := start + i; idx < len(lines) {
			windowed[i] = lines[idx]
		}
	}
	return strings.Join(windowed, "\n")
}

// rowView renders one Project row: NAME  STATUS  LAST VISIT, with a caret
// and accent name when selected.
func (m Model) rowView(t theme, mt match, selected bool, lay Layout, now time.Time) string {
	caret := "  "
	nameStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		caret = t.fg(t.accent).Bold(true).Render("›")
		nameStyle = nameStyle.Foreground(t.accent)
	}

	name := mt.row.Project.Name
	if len([]rune(name)) > lay.NameWidth {
		name = truncateName(name, lay.NameWidth)
	}
	offset := len([]rune(mt.row.Project.Path)) - len([]rune(mt.row.Project.Name))
	name = padRight(highlightMatches(name, mt.matches, offset, nameStyle, t), lay.NameWidth)

	st, loaded := m.statuses[mt.row.Project.Path]
	status := padRight(t.statusCluster(st, loaded), lay.StatusWidth)

	rel := RelativeTime(mt.row.LastVisit, now)
	if lay.ShortTime {
		rel = RelativeTimeShort(mt.row.LastVisit, now)
	}
	rel = padLeft(t.muted_().Render(rel), lay.TimeWidth)

	return caret + " " + name + "  " + status + "  " + rel
}

// padRight/padLeft pad plain or styled strings to a display width.
func padRight(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func padLeft(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		return strings.Repeat(" ", d) + s
	}
	return s
}

// previewView renders the right-hand preview box for the selected row.
func (m Model) previewView(t theme, rows []match, lay Layout, now time.Time) string {
	var body strings.Builder
	if m.cursor >= 0 && m.cursor < len(rows) {
		p := rows[m.cursor].row.Project
		st, loaded := m.statuses[p.Path]

		label := func(s string) string { return t.muted_().Render(padRight(s, 11)) }
		body.WriteString(t.muted_().Render(p.Kind+"/") + t.accentBold().Render(p.Name) + "\n\n")
		body.WriteString(label("path") + p.Path + "\n")
		if loaded && st.Kind == git.Found {
			body.WriteString(label("branch") + st.Branch + "\n")
		}
		body.WriteString(label("status") + previewStatusWords(t, st, loaded) + "\n")
		if sync := previewSync(t, st, loaded); sync != "" {
			body.WriteString(label("sync") + sync + "\n")
		}
		last := rows[m.cursor].row.LastVisit
		if last.IsZero() {
			body.WriteString(label("last visit") + t.muted_().Render("never"))
		} else {
			body.WriteString(label("last visit") + last.Format("2006-01-02 15:04"))
			body.WriteString(t.muted_().Render("  " + RelativeTime(last, now)))
		}
		body.WriteString("\n" + label("visits") + fmt.Sprintf("%d", rows[m.cursor].row.Visits))
	} else {
		body.WriteString(t.muted_().Render("nothing selected"))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.rule).
		Padding(0, 1).
		Width(lay.PreviewWidth).
		Height(lay.ListHeight)
	return box.Render(body.String())
}

// footerView renders the rule, legend line (when there is room), keys line
// (when there is room) and the match count.
func (m Model) footerView(t theme, width, matched int, lay Layout) string {
	var lines []string
	lines = append(lines, t.rule_(width))
	if lay.ShowLegend {
		lines = append(lines, t.legend())
	}
	if lay.ShowKeys {
		keys := t.keysLine("↑↓", "move", "enter", "jump", "esc", "cancel", "ctrl+u", "clear")
		if m.vim {
			keys = t.keysLine("j/k", "move", "f", "filter", "esc", "back/cancel", "enter", "jump", "q", "quit")
		}
		count := t.muted_().Render(fmt.Sprintf("%d/%d", matched, len(m.rows)))
		gap := max(width-lipgloss.Width(keys)-lipgloss.Width(count), 1)
		lines = append(lines, keys+strings.Repeat(" ", gap)+count)
	}
	return strings.Join(lines, "\n")
}
