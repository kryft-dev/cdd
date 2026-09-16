package main

// PROTOTYPE variant D: the composite the human asked for after seeing A, B
// and C. B's grouping by Kind and caret selection, A's glyph cluster with
// ahead/behind counts and the footer legend, C's preview pane.

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func viewComposite(m model, t theme, w, h int) string {
	rows := m.visible()
	nameW, statusW := 8, 1
	for _, r := range rows {
		nameW = max(nameW, len(r.p.Name))
		statusW = max(statusW, statusWidth(*r.p))
	}
	leftW := min(max(3+nameW+2+statusW+2+10, 36), w*55/100)
	rightW := w - leftW - 1
	listH := h - 4 // filter, rule, legend, keys

	var order []string
	byKind := map[string][]int{}
	for i, r := range rows {
		if _, ok := byKind[r.p.Kind]; !ok {
			order = append(order, r.p.Kind)
		}
		byKind[r.p.Kind] = append(byKind[r.p.Kind], i)
	}

	// Left: grouped list.
	var lines []string
	for _, kind := range order {
		head := t.fg(t.purple).Bold(true).Render(" "+kind+" ") + t.fg(t.rule).Render(strings.Repeat("─", max(leftW-len(kind)-3, 0)))
		lines = append(lines, head)
		for _, i := range byKind[kind] {
			r := rows[i]
			base := lipgloss.NewStyle()
			caret := "   "
			nameStyle := base.Bold(true)
			if i == m.cursor {
				caret = t.fg(t.blue).Bold(true).Render(" › ")
				nameStyle = nameStyle.Foreground(t.blue)
			}
			name := highlight(pad(r.p.Name, nameW), r.matched, len(r.p.Kind)+1, nameStyle, t.Match(nameStyle))
			status := t.statusCell(*r.p, base) + strings.Repeat(" ", statusW-statusWidth(*r.p))
			last := t.Muted().Render(relTime(r.p.Last))
			left := caret + name + "  " + status
			lines = append(lines, left+strings.Repeat(" ", max(leftW-lipgloss.Width(left)-lipgloss.Width(last)-1, 1))+last)
		}
	}
	if len(rows) == 0 {
		lines = append(lines, t.Muted().Render("   no projects match"))
	}
	// Keep the cursor's line on screen.
	cursorLine := 0
	for n, kind := range order {
		for _, i := range byKind[kind] {
			if i == m.cursor {
				cursorLine = n + 1 + i
			}
		}
	}
	start := 0
	if cursorLine >= listH {
		start = cursorLine - listH + 1
	}
	var lb strings.Builder
	lb.WriteString(m.filterLine(t, " ❯ ") + "\n")
	for i := start; i < start+listH; i++ {
		if i < len(lines) {
			lb.WriteString(lines[i])
		}
		lb.WriteString("\n")
	}
	left := strings.TrimRight(lb.String(), "\n")

	// Right: preview of the selected Project, under a matching blank line.
	var rb strings.Builder
	if len(rows) > 0 {
		p := rows[m.cursor].p
		label := func(s string) string { return t.Muted().Render(pad(s, 11)) }
		rb.WriteString(t.Muted().Render(p.Kind+"/") + t.fg(t.purple).Bold(true).Render(p.Name) + "\n\n")
		rb.WriteString(label("path") + p.Path() + "\n")
		if p.Repo {
			rb.WriteString(label("branch") + p.Branch + "\n")
		}
		status, sync := previewStatus(t, *p)
		rb.WriteString(label("status") + status + "\n")
		if sync != "" {
			rb.WriteString(label("sync") + sync + "\n")
		}
		rb.WriteString(label("last visit") + absTime(p.Last))
		if rel := relTime(p.Last); rel != "" {
			rb.WriteString(t.Muted().Render("  " + rel))
		}
		rb.WriteString("\n" + label("visits") + plural(p.Visits, "jump"))
	} else {
		rb.WriteString(t.Muted().Render("nothing selected"))
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.rule).Padding(0, 1).Width(rightW).Height(listH)
	right := "\n" + box.Render(rb.String())

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right) + "\n"
	keys := t.keys("↑↓", "move", "enter", "jump", "esc", "cancel", "ctrl+u", "clear")
	if m.keymap == "vim" {
		keys = t.keys("j/k", "move", "f", "filter", "esc", "back / cancel", "enter", "jump", "q", "quit")
	}
	count := t.Muted().Render(fmt.Sprintf("%d/%d", len(rows), len(m.projects)))
	return body + t.Rule(w) + "\n " + t.legend() + "\n " + keys + strings.Repeat(" ", max(w-lipgloss.Width(keys)-lipgloss.Width(count)-2, 1)) + count
}

// previewStatus splits the spelled-out status into a working-tree line and
// a sync line so neither wraps inside the preview box.
func previewStatus(t theme, p project) (status, sync string) {
	f := func(c interface{ RGBA() (r, g, b, a uint32) }, s string) string { return t.fg(c).Render(s) }
	switch {
	case !p.Repo:
		return f(t.muted, gNoRepo+" not a repository"), ""
	case !p.Loaded:
		return f(t.muted, gPending+" loading"), ""
	case p.Timeout:
		return f(t.red, gTimeout+" git timed out"), ""
	}
	if p.Dirty {
		status = f(t.yellow, gDirty+" modified")
	} else {
		status = f(t.green, gClean+" clean")
	}
	if p.Untracked {
		status += "  " + f(t.blue, gUntracked+" untracked files")
	}
	var parts []string
	if p.Ahead > 0 {
		parts = append(parts, f(t.purple, fmt.Sprintf("%s%d ahead", gAhead, p.Ahead)))
	}
	if p.Behind > 0 {
		parts = append(parts, f(t.purple, fmt.Sprintf("%s%d behind", gBehind, p.Behind)))
	}
	if !p.Upstream {
		parts = append(parts, t.Muted().Render("no upstream"))
	}
	if len(parts) == 0 {
		parts = append(parts, t.Muted().Render("in sync"))
	}
	return status, strings.Join(parts, "  ")
}
