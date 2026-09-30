package picker

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// selBar marks the selected row in the list layout.
const selBar = "▌"

// listFrame renders the flat fzf-style layout: rows in the order they were
// given (History order), the filter prompt below them where fzf users
// expect it, and the shared preview pane and footer.
func (m Model) listFrame(t theme, now time.Time) string {
	rows := m.visibleRows()
	width, height := m.frameSize()
	lay := m.computeLayout(rows, now, width, height)

	var b strings.Builder
	body := m.listBody(t, rows, lay, now)
	if lay.ShowPreview {
		// The preview box is exactly ListHeight lines tall, matching the
		// body, so the frame stays at the terminal height.
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, body, " ", m.previewView(t, rows, lay, now)))
	} else {
		b.WriteString(body)
	}

	b.WriteString("\n")
	b.WriteString(m.filterLine(t))
	b.WriteString("\n")
	b.WriteString(m.footerView(t, width, len(rows), lay))

	return b.String()
}

// listBody renders the list pane: one line per visible row, windowed to
// exactly Layout.ListHeight lines and scrolled to keep the cursor's row on
// screen.
func (m Model) listBody(t theme, rows []match, lay Layout, now time.Time) string {
	lines := make([]string, 0, max(len(rows), 1))
	for i, mt := range rows {
		lines = append(lines, m.listRowView(t, mt, i == m.cursor, lay, now))
	}
	if len(rows) == 0 {
		lines = append(lines, t.muted_().Render("no projects match"))
	}

	return window(lines, m.cursor, lay.ListHeight)
}

// listRowView renders one row as "▌ STATUS kind/NAME   LAST VISIT", padded
// out to Layout.ListWidth. Every segment, padding included, is rendered
// through base, so the selected row's background highlight runs unbroken
// to the edge of the pane.
func (m Model) listRowView(t theme, mt match, selected bool, lay Layout, now time.Time) string {
	base := lipgloss.NewStyle()
	nameStyle := base
	bar := base.Render(" ")
	if selected {
		base = base.Background(t.selBg)
		nameStyle = base.Foreground(t.selFg).Bold(true)
		bar = base.Foreground(t.blue).Render(selBar)
	}
	kindStyle := base.Foreground(t.muted)

	st, loaded := m.statuses[mt.row.Project.Path]
	status := padRightOn(base, t.statusClusterOn(base, st, loaded), lay.StatusWidth)
	name := m.listNameField(t, mt, lay, base, kindStyle, nameStyle)

	rel := RelativeTime(mt.row.LastVisit, now)
	if lay.ShortTime {
		rel = RelativeTimeShort(mt.row.LastVisit, now)
	}
	rel = padLeftOn(base, base.Foreground(t.muted).Render(rel), lay.TimeWidth)

	lead := bar + base.Render(" ") + status + base.Render(" ") + name
	gap := max(lay.ListWidth-lipgloss.Width(lead)-lipgloss.Width(rel)-1, 1)
	return lead + base.Render(strings.Repeat(" ", gap)) + rel + base.Render(" ")
}

// listNameFloor is the least room the Project name keeps in the name
// column. A Kind long enough to crowd it out is truncated instead: the
// name is what the user is reading for.
const listNameFloor = 4

// listNameField renders "kind/name" padded to Layout.NameWidth, with the
// Kind muted ahead of the Project name and any fuzzy-match runes
// highlighted. The Project name is truncated first when the pair is too
// wide, and the Kind once the name is down to listNameFloor. Each segment
// keeps its own style, so a long Kind never mutes the name with it.
func (m Model) listNameField(t theme, mt match, lay Layout, base, kindStyle, nameStyle lipgloss.Style) string {
	p := mt.row.Project
	kind := p.Kind + "/"
	name := p.Name

	// Matched indexes are rune offsets into the Project's path, of which
	// "kind/name" is the tail; shift them onto each segment.
	pathLen := len([]rune(p.Path))
	kindOffset := pathLen - len([]rune(kind)) - len([]rune(name))
	nameOffset := pathLen - len([]rune(name))

	kindWidth := len([]rune(kind))
	if over := kindWidth - (lay.NameWidth - listNameFloor); over > 0 {
		kindWidth = max(kindWidth-over, 0)
		kind = truncateName(kind, kindWidth)
	}
	name = truncateName(name, max(lay.NameWidth-kindWidth, 0))

	field := highlightMatches(kind, mt.matches, kindOffset, kindStyle, t) +
		highlightMatches(name, mt.matches, nameOffset, nameStyle, t)
	return padRightOn(base, field, lay.NameWidth)
}
