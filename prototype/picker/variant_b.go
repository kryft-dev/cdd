package main

// PROTOTYPE variant B: grouped by Kind. Kind is a section header rather than
// a column, status is spelled out inline so the footer carries no legend, and
// the selected row is marked by a caret and colour instead of a background.

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func viewGrouped(m model, t theme, w, h int) string {
	rows := m.visible()
	nameW, branchW := 8, 6
	for _, r := range rows {
		nameW = max(nameW, len(r.p.Name))
		branchW = max(branchW, min(len(r.p.Branch), 16))
	}
	var order []string
	byKind := map[string][]int{}
	for i, r := range rows {
		if _, ok := byKind[r.p.Kind]; !ok {
			order = append(order, r.p.Kind)
		}
		byKind[r.p.Kind] = append(byKind[r.p.Kind], i)
	}

	var b strings.Builder
	b.WriteString(m.filterLine(t, " ❯ ") + "\n")
	lines := 1
	avail := h - 3
	for _, kind := range order {
		if lines >= avail {
			break
		}
		head := t.fg(t.purple).Bold(true).Render(" "+kind+" ") + t.fg(t.rule).Render(strings.Repeat("─", max(w-len(kind)-3, 0)))
		b.WriteString(head + "\n")
		lines++
		for _, i := range byKind[kind] {
			if lines >= avail {
				b.WriteString(t.Muted().Render("   …") + "\n")
				lines++
				break
			}
			r := rows[i]
			base := lipgloss.NewStyle()
			caret := "   "
			nameStyle := base.Bold(true)
			if i == m.cursor {
				caret = t.fg(t.blue).Bold(true).Render(" › ")
				nameStyle = nameStyle.Foreground(t.blue)
			}
			name := highlight(pad(r.p.Name, nameW), r.matched, len(r.p.Kind)+1, nameStyle, t.Match(nameStyle))
			branch := r.p.Branch
			if len(branch) > 16 {
				branch = branch[:15] + "…"
			}
			last := t.Muted().Render(relTime(r.p.Last))
			left := caret + name + "  " + t.Muted().Render(pad(branch, branchW)) + "  " + t.statusWords(*r.p, base)
			line := left + strings.Repeat(" ", max(w-lipgloss.Width(left)-lipgloss.Width(last)-1, 1)) + last
			b.WriteString(line + "\n")
			lines++
		}
	}
	if len(rows) == 0 {
		b.WriteString(t.Muted().Render("   no projects match") + "\n")
		lines++
	}
	for ; lines < avail; lines++ {
		b.WriteString("\n")
	}
	b.WriteString(t.Rule(w) + "\n")
	b.WriteString(" " + t.keys("↑↓", "move", "enter", "jump", "esc", "cancel", "ctrl+u", "clear") + "   " + t.Muted().Render(plural(len(rows), "project")+" in "+plural(len(order), "kind")))
	return b.String()
}
