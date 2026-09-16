package main

// PROTOTYPE variant A: gh-dash style. Filter on top, column headers, an
// aligned table with a full-width highlighted row, glyph legend in the footer.

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func viewTable(m model, t theme, w, h int) string {
	rows := m.visible()
	nameW, kindW, branchW, statusW := 8, 4, 6, 6
	for _, r := range rows {
		nameW = max(nameW, len(r.p.Name))
		kindW = max(kindW, len(r.p.Kind))
		branchW = max(branchW, min(len(r.p.Branch), 18))
		statusW = max(statusW, statusWidth(*r.p))
	}
	timeW := 10
	gap := "  "

	var b strings.Builder
	count := t.Muted().Render(plural(len(rows), "project"))
	filter := m.filterLine(t, "  ")
	b.WriteString(filter + strings.Repeat(" ", max(w-lipgloss.Width(filter)-lipgloss.Width(count), 1)) + count + "\n")
	b.WriteString(t.Header().Render("   "+pad("PROJECT", nameW)+gap+pad("KIND", kindW)+gap+pad("BRANCH", branchW)+gap+pad("STATUS", statusW)+gap+"LAST VISIT") + "\n")

	avail := h - 5
	for i, r := range rows {
		if i >= avail {
			b.WriteString(t.Muted().Render("   … "+plural(len(rows)-avail, "more")) + "\n")
			break
		}
		base := lipgloss.NewStyle()
		marker := "   "
		if i == m.cursor {
			base = base.Background(t.selBg)
			marker = base.Foreground(t.selFg).Bold(true).Render(" ▶ ")
		} else {
			marker = base.Render(marker)
		}
		nameStyle := base
		if i == m.cursor {
			nameStyle = base.Foreground(t.selFg).Bold(true)
		}
		off := len(r.p.Kind) + 1
		name := highlight(pad(r.p.Name, nameW), r.matched, off, nameStyle, t.Match(nameStyle))
		kind := highlight(pad(r.p.Kind, kindW), r.matched, 0, base.Foreground(t.muted), t.Match(base))
		branch := r.p.Branch
		if len(branch) > 18 {
			branch = branch[:17] + "…"
		}
		branch = base.Foreground(t.muted).Render(pad(branch, branchW))
		status := t.statusCell(*r.p, base) + base.Render(strings.Repeat(" ", statusW-statusWidth(*r.p)))
		last := base.Foreground(t.muted).Render(pad(relTime(r.p.Last), timeW))
		line := marker + name + base.Render(gap) + kind + base.Render(gap) + branch + base.Render(gap) + status + base.Render(gap) + last
		if rem := w - lipgloss.Width(line); rem > 0 {
			line += base.Render(strings.Repeat(" ", rem))
		}
		b.WriteString(line + "\n")
	}
	for i := len(rows); i < avail; i++ {
		b.WriteString("\n")
	}
	if len(rows) == 0 {
		b.WriteString(t.Muted().Render("   no projects match") + "\n")
	}
	b.WriteString(t.Rule(w) + "\n")
	b.WriteString(" " + t.legend() + "\n")
	b.WriteString(" " + t.keys("↑↓", "move", "enter", "jump", "esc", "cancel", "ctrl+u", "clear"))
	return b.String()
}
