package main

// PROTOTYPE variant C: fzf-style list with a preview pane. Prompt at the
// bottom of the list, one glyph per row, and a boxed detail pane on the right
// that spells everything out for the selected Project.

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func viewSplit(m model, t theme, w, h int) string {
	rows := m.visible()
	listH := h - 4 // prompt line, rule, legend, keys
	statusW, keyW := 1, 8
	for _, r := range rows {
		statusW = max(statusW, statusWidth(*r.p))
		keyW = max(keyW, len(r.p.Key()))
	}
	// Left pane sized to content, capped at just over half the screen; keys
	// that still don't fit are truncated.
	leftW := min(max(3+statusW+keyW+12, 32), w*55/100)
	keyW = min(keyW, leftW-3-statusW-12)
	rightW := w - leftW - 1

	// Left: list, most recent first, prompt below like fzf.
	var lb strings.Builder
	start := 0
	if m.cursor >= listH {
		start = m.cursor - listH + 1
	}
	for i := start; i < len(rows) && i < start+listH; i++ {
		r := rows[i]
		base := lipgloss.NewStyle()
		bar := " "
		if i == m.cursor {
			base = base.Background(t.selBg)
			bar = base.Foreground(t.blue).Render("▌")
		} else {
			bar = base.Render(bar)
		}
		nameStyle := base
		if i == m.cursor {
			nameStyle = base.Foreground(t.selFg).Bold(true)
		}
		off := len(r.p.Kind) + 1
		kind := highlight(r.p.Kind+"/", r.matched, 0, base.Foreground(t.muted), t.Match(base))
		nm := r.p.Name
		if over := len(r.p.Key()) - keyW; over > 0 {
			nm = nm[:len(nm)-over-1] + "…"
		}
		name := highlight(nm, r.matched, off, nameStyle, t.Match(nameStyle))
		last := base.Foreground(t.muted).Render(relTime(r.p.Last))
		glyph := t.statusCell(*r.p, base)
		lead := bar + base.Render(" ") + glyph + base.Render(strings.Repeat(" ", statusW+1-statusWidth(*r.p))) + kind + name
		fill := leftW - lipgloss.Width(lead) - lipgloss.Width(last) - 1
		lb.WriteString(lead + base.Render(strings.Repeat(" ", max(fill, 1))) + last + base.Render(" ") + "\n")
	}
	for i := len(rows) - start; i < listH; i++ {
		lb.WriteString("\n")
	}
	count := t.Muted().Render(fmt.Sprintf("%d/%d", len(rows), len(m.projects)))
	prompt := m.filterLine(t, " > ")
	lb.WriteString(prompt + strings.Repeat(" ", max(leftW-lipgloss.Width(prompt)-lipgloss.Width(count), 1)) + count)
	left := lb.String()

	// Right: preview of the selected Project.
	var rb strings.Builder
	if len(rows) > 0 {
		p := rows[m.cursor].p
		label := func(s string) string { return t.Muted().Render(pad(s, 11)) }
		rb.WriteString(t.fg(t.purple).Bold(true).Render(p.Key()) + "\n\n")
		rb.WriteString(label("path") + p.Path() + "\n")
		if p.Repo {
			rb.WriteString(label("branch") + p.Branch + "\n")
		}
		rb.WriteString(label("status") + t.statusWords(*p, lipgloss.NewStyle()) + "\n")
		rb.WriteString(label("last visit") + absTime(p.Last))
		if rel := relTime(p.Last); rel != "" {
			rb.WriteString(t.Muted().Render("  " + rel))
		}
		rb.WriteString("\n" + label("visits") + plural(p.Visits, "jump"))
	} else {
		rb.WriteString(t.Muted().Render("nothing selected"))
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.rule).Padding(0, 1).Width(rightW).Height(listH)
	right := box.Render(rb.String())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return body + "\n" + t.Rule(w) + "\n " + t.legend() + "\n " + t.keys("↑↓", "move", "enter", "jump", "esc", "cancel")
}
