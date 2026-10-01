package picker

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/git"
)

// View renders the current frame in the Model's layout. The layout's own
// frame draws the list; the preview pane and footer here are shared by any
// layout.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	// Nothing is drawn until the terminal has reported its background
	// colour or paletteDeadline has passed: a frame drawn in the wrong
	// palette repaints in the right one a moment later, which is the
	// flash a user sees on launch.
	if !m.paletteSettled {
		return fullScreen("")
	}
	if len(m.rows) == 0 {
		return fullScreen(m.emptyHistoryView())
	}

	t := newTheme(m.dark)
	now := time.Now()
	return fullScreen(m.listFrame(t, now))
}

// fullScreen wraps content in a View drawn on the alternate screen. The
// frame always fills the terminal, and the alternate screen guarantees
// the shell's own scrollback comes back untouched when the Picker exits;
// inline rendering left stray lines above the prompt.
func fullScreen(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
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

// previewView renders the right-hand preview box for the selected row.
func (m Model) previewView(t theme, rows []hit, met Metrics, now time.Time) string {
	var body strings.Builder
	if m.cursor >= 0 && m.cursor < len(rows) {
		p := rows[m.cursor].row.Project
		st, loaded := m.statuses[p.Path]

		label := func(s string) string { return t.muted_().Render(padRightOn(plainStyle, s, 11)) }
		body.WriteString(t.muted_().Render(p.Dir) + t.accentBold().Render(p.Name) + "\n\n")
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
		Width(met.PreviewWidth).
		Height(met.ListHeight).
		MaxHeight(met.ListHeight) // Height is a minimum; a tall body must not grow the box
	return box.Render(body.String())
}

// footerView renders the rule, legend line (when there is room), keys line
// (when there is room) and the match count.
func (m Model) footerView(t theme, width, matched int, met Metrics) string {
	var lines []string
	lines = append(lines, t.rule_(width))
	if met.ShowLegend {
		lines = append(lines, t.legend())
	}
	if met.ShowKeys {
		keys := t.keysLine("↑↓", "move", "enter", "jump", "esc", "cancel", "ctrl+u", "clear")
		if m.vim {
			keys = t.keysLine("j/k", "move", "f", "filter", "esc", "back/cancel", "enter", "jump", "q", "quit")
		}
		count := t.muted_().Render(fmt.Sprintf("%d/%d", matched, len(m.rows)))
		gap := max(width-lipgloss.Width(keys)-lipgloss.Width(count), 1)
		if m.message != "" {
			colour := t.red
			if m.messageOK {
				colour = t.green
			}
			keys = t.fg(colour).Render(truncateName(m.message, max(width-lipgloss.Width(count)-1, 1)))
			gap = max(width-lipgloss.Width(keys)-lipgloss.Width(count), 1)
		}
		lines = append(lines, keys+strings.Repeat(" ", gap)+count)
	}
	return strings.Join(lines, "\n")
}
