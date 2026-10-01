package picker

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// hintSepWidth is the width of the " · " between two key hints.
const hintSepWidth = 3

// footerView renders the rule, legend line (when there is room), keys line
// (when there is room) and the match count.
func (m Model) footerView(t theme, width, matched int, met Metrics) string {
	var lines []string
	lines = append(lines, t.rule_(width))
	if met.ShowLegend {
		lines = append(lines, t.legend())
	}
	if met.ShowKeys {
		lines = append(lines, m.keysLine(t, width, matched))
	}
	return strings.Join(lines, "\n")
}

// keysLine is the footer's one line: on the left the message, else the key
// hints, and on the right the match count, with a pointer to the help
// overlay ahead of it where `?` opens one. A message replaces the hints
// until the next key press.
func (m Model) keysLine(t theme, width, matched int) string {
	right := t.muted_().Render(fmt.Sprintf("%d/%d", matched, len(m.rows)))
	if m.helpAvailable() && !m.noHints {
		right = t.keysLine("?", "help") + "  " + right
	}
	room := max(width-lipgloss.Width(right)-1, 1)

	var left string
	switch {
	case m.message != "":
		left = t.messageLine(m.message, m.messageOK, room)
	case !m.noHints:
		left = t.hintsLine(m.keyHints(), room)
	}
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

// keyHints lists the keys that run an Action right now, each with its
// name, as (key, name) pairs in the order the Actions came in. Unbound
// Actions are left out, and so are the printable keys the filter has
// taken for typing.
func (m Model) keyHints() [][2]string {
	var out [][2]string
	for _, a := range m.hints {
		if _, ok := m.boundAction(a.Key); ok {
			out = append(out, [2]string{a.Key, a.Name})
		}
	}
	return out
}

// messageLine renders a one-line message in green when it is good news and
// red when it is an error, cut to width.
func (t theme) messageLine(text string, ok bool, width int) string {
	colour := t.red
	if ok {
		colour = t.green
	}
	return t.fg(colour).Render(truncateName(text, width))
}

// hintsLine renders (key, name) pairs like keysLine: as many whole pairs as
// fit in width, followed by " …" when some are dropped. When not even the
// first fits, it is cut to width instead.
func (t theme) hintsLine(items [][2]string, width int) string {
	if len(items) == 0 {
		return ""
	}
	widthOf := func(n int) int {
		w := 0
		for i, it := range items[:n] {
			w += lipgloss.Width(it[0] + " " + it[1])
			if i > 0 {
				w += hintSepWidth
			}
		}
		return w
	}

	n := len(items)
	for n > 0 && widthOf(n) > width {
		n--
	}
	if n < len(items) {
		// Dropping any leaves room to say so.
		for n > 0 && widthOf(n)+2 > width {
			n--
		}
	}
	if n == 0 {
		return t.muted_().Render(truncateName(items[0][0]+" "+items[0][1], width))
	}

	flat := make([]string, 0, 2*n)
	for _, it := range items[:n] {
		flat = append(flat, it[0], it[1])
	}
	line := t.keysLine(flat...)
	if n < len(items) {
		line += t.muted_().Render(" …")
	}
	return line
}
