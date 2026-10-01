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
	met := m.computeMetrics(rows, now, width, height)

	var b strings.Builder
	body := m.listBody(t, rows, met, now)
	if met.ShowPreview {
		// The preview box is exactly ListHeight lines tall, matching the
		// body, so the frame stays at the terminal height.
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, body, " ", m.previewView(t, rows, met, now)))
	} else {
		b.WriteString(body)
	}

	b.WriteString("\n")
	b.WriteString(m.filterLine(t))
	b.WriteString("\n")
	b.WriteString(m.footerView(t, width, len(rows), met))

	return b.String()
}

// listBody renders the list pane: one line per visible row, windowed to
// exactly Metrics.ListHeight lines and scrolled to keep the cursor's row on
// screen.
func (m Model) listBody(t theme, rows []hit, met Metrics, now time.Time) string {
	lines := make([]string, 0, max(len(rows), 1))
	for i, mt := range rows {
		lines = append(lines, m.listRowView(t, mt, i == m.cursor, met, now))
	}
	if len(rows) == 0 {
		lines = append(lines, t.muted_().Render("no projects match"))
	}

	return window(lines, m.cursor, met.ListHeight)
}

// listRowView renders one row as "▌ STATUS ~/dir/NAME   LAST VISIT", padded
// out to Metrics.ListWidth. Every segment, padding included, is rendered
// through base, so the selected row's background highlight runs unbroken
// to the edge of the pane.
func (m Model) listRowView(t theme, mt hit, selected bool, met Metrics, now time.Time) string {
	base := lipgloss.NewStyle()
	nameStyle := base
	bar := base.Render(" ")
	if selected {
		base = base.Background(t.selBg)
		nameStyle = base.Foreground(t.selFg).Bold(true)
		bar = base.Foreground(t.blue).Render(selBar)
	}
	dirStyle := base.Foreground(t.muted)

	st, loaded := m.statuses[mt.row.Project.Path]
	status := padRightOn(base, t.statusClusterOn(base, st, loaded), met.StatusWidth)
	name := m.listNameField(t, mt, met, base, dirStyle, nameStyle)

	rel := RelativeTime(mt.row.LastVisit, now)
	if met.ShortTime {
		rel = RelativeTimeShort(mt.row.LastVisit, now)
	}
	rel = padLeftOn(base, base.Foreground(t.muted).Render(rel), met.TimeWidth)

	lead := bar + base.Render(" ") + status + base.Render(" ") + name
	gap := max(met.ListWidth-lipgloss.Width(lead)-lipgloss.Width(rel)-1, 1)
	return lead + base.Render(strings.Repeat(" ", gap)) + rel + base.Render(" ")
}

// listNameFloor is the least room the Project name keeps in the name
// column. A parent directory long enough to crowd it out is truncated
// instead: the name is what the user is reading for.
const listNameFloor = 4

// listNameField renders the Project's parent directory and name padded to
// Metrics.NameWidth, the directory muted ahead of the name and any
// matched runes highlighted. The name is truncated first when the pair
// is too wide, and the directory, from its start, once the name is down to
// listNameFloor. Each segment keeps its own style, so a long directory
// never mutes the name with it.
func (m Model) listNameField(t theme, mt hit, met Metrics, base, dirStyle, nameStyle lipgloss.Style) string {
	p := mt.row.Project
	dir, name := p.Dir, p.Name

	// Matched indexes are rune offsets into Dir+Name; shift them onto each
	// segment, and past whatever truncateLeft drops from the directory.
	dirWidth := len([]rune(dir))
	nameOffset := dirWidth
	dirOffset := 0
	if over := dirWidth - (met.NameWidth - listNameFloor); over > 0 {
		kept := max(dirWidth-over, 0)
		dir = truncateLeft(dir, kept)
		dirOffset = dirWidth - kept
		dirWidth = kept
	}
	name = truncateName(name, max(met.NameWidth-dirWidth, 0))

	field := highlightMatches(dir, mt.matches, dirOffset, dirStyle, t) +
		highlightMatches(name, mt.matches, nameOffset, nameStyle, t)
	return padRightOn(base, field, met.NameWidth)
}
