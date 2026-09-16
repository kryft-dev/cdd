package picker

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/git"
)

// theme holds the accent colours the Picker needs, chosen for the
// terminal's background via lipgloss.LightDark. Body text keeps the
// terminal's own foreground.
type theme struct {
	muted, green, yellow, blue, red, accent, rule color.Color
}

// newTheme builds the theme for a light or dark background, using the
// colours settled on for the accepted look.
func newTheme(dark bool) theme {
	ld := lipgloss.LightDark(dark)
	c := lipgloss.Color
	return theme{
		muted:  ld(c("#6E7781"), c("#8B949E")),
		green:  ld(c("#1A7F37"), c("#3FB950")),
		yellow: ld(c("#9A6700"), c("#D29922")),
		blue:   ld(c("#0969DA"), c("#58A6FF")),
		red:    ld(c("#CF222E"), c("#F85149")),
		accent: ld(c("#8250DF"), c("#A371F7")),
		rule:   ld(c("#D0D7DE"), c("#30363D")),
	}
}

func (t theme) fg(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
func (t theme) muted_() lipgloss.Style          { return t.fg(t.muted) }
func (t theme) accentBold() lipgloss.Style      { return t.fg(t.accent).Bold(true) }
func (t theme) match() lipgloss.Style           { return t.fg(t.accent).Bold(true).Underline(true) }
func (t theme) rule_(w int) string {
	if w < 0 {
		w = 0
	}
	return t.fg(t.rule).Render(strings.Repeat("─", w))
}

// Status glyphs, shared by the row cluster, the preview pane and the
// legend.
const (
	glyphClean     = "✓"
	glyphModified  = "●"
	glyphUntracked = "?"
	glyphAhead     = "↑"
	glyphBehind    = "↓"
	glyphUnknown   = "!"
	glyphNotRepo   = "—"
	glyphLoading   = "…"
)

// statusCluster renders the compact coloured glyph cluster for one row's
// status: "✓", "● ?", "● ↑3↓2", "—", "…", "!".
func (t theme) statusCluster(st git.Status, loaded bool) string {
	if !loaded {
		return t.muted_().Render(glyphLoading)
	}
	switch st.Kind {
	case git.NotRepo:
		return t.muted_().Render(glyphNotRepo)
	case git.Unknown:
		return t.fg(t.red).Render(glyphUnknown)
	}

	var parts []string
	if st.State == git.Dirty {
		parts = append(parts, t.fg(t.yellow).Render(glyphModified))
	} else {
		parts = append(parts, t.fg(t.green).Render(glyphClean))
	}
	if st.Untracked {
		parts = append(parts, t.fg(t.blue).Render(glyphUntracked))
	}
	sync := ""
	if st.Ahead > 0 {
		sync += glyphAhead + strconv.Itoa(st.Ahead)
	}
	if st.Behind > 0 {
		sync += glyphBehind + strconv.Itoa(st.Behind)
	}
	if sync != "" {
		parts = append(parts, t.fg(t.accent).Render(sync))
	}
	return strings.Join(parts, " ")
}

// statusClusterWidth is the plain (uncoloured) width of statusCluster's
// output, for column alignment.
func statusClusterWidth(st git.Status, loaded bool) int {
	if !loaded || st.Kind != git.Found {
		return 1
	}
	w := 1
	if st.Untracked {
		w += 2
	}
	sync := 0
	if st.Ahead > 0 {
		sync += 1 + len(strconv.Itoa(st.Ahead))
	}
	if st.Behind > 0 {
		sync += 1 + len(strconv.Itoa(st.Behind))
	}
	if sync > 0 {
		w += 1 + sync
	}
	return w
}

// legend is the plain-word footer key for the status glyphs.
func (t theme) legend() string {
	f := func(c color.Color, glyph, word string) string {
		return t.fg(c).Render(glyph) + t.muted_().Render(" "+word)
	}
	return strings.Join([]string{
		f(t.green, glyphClean, "clean"),
		f(t.yellow, glyphModified, "modified"),
		f(t.blue, glyphUntracked, "untracked"),
		f(t.accent, glyphAhead, "ahead"),
		f(t.accent, glyphBehind, "behind"),
		f(t.red, glyphUnknown, "unknown"),
		f(t.muted, glyphNotRepo, "not a repo"),
		f(t.muted, glyphLoading, "loading"),
	}, "  ")
}

// keysLine renders a key/description legend, e.g. "↑↓ move · enter jump".
func (t theme) keysLine(items ...string) string {
	var parts []string
	for i := 0; i+1 < len(items); i += 2 {
		parts = append(parts, t.fg(t.muted).Bold(true).Render(items[i])+t.muted_().Render(" "+items[i+1]))
	}
	return strings.Join(parts, t.muted_().Render(" · "))
}

// previewStatusWords spells the working-tree status out in words, for the
// preview pane's "status" line.
func previewStatusWords(t theme, st git.Status, loaded bool) string {
	if !loaded {
		return t.muted_().Render(glyphLoading + " loading")
	}
	switch st.Kind {
	case git.NotRepo:
		return t.muted_().Render(glyphNotRepo + " not a repository")
	case git.Unknown:
		return t.fg(t.red).Render(glyphUnknown + " git timed out")
	}

	var parts []string
	if st.State == git.Dirty {
		parts = append(parts, t.fg(t.yellow).Render(glyphModified+" modified"))
	} else {
		parts = append(parts, t.fg(t.green).Render(glyphClean+" clean"))
	}
	if st.Untracked {
		parts = append(parts, t.fg(t.blue).Render(glyphUntracked+" untracked files"))
	}
	return strings.Join(parts, "  ")
}

// previewSync spells the ahead/behind/upstream state out, for the preview
// pane's "sync" line. It is "" when there is nothing to report (not a
// repository, unknown, or still loading).
func previewSync(t theme, st git.Status, loaded bool) string {
	if !loaded || st.Kind != git.Found {
		return ""
	}
	var parts []string
	if st.Ahead > 0 {
		parts = append(parts, t.fg(t.accent).Render(strconv.Itoa(st.Ahead)+" ahead"))
	}
	if st.Behind > 0 {
		parts = append(parts, t.fg(t.accent).Render(strconv.Itoa(st.Behind)+" behind"))
	}
	if !st.HasUpstream {
		return t.muted_().Render("no upstream")
	}
	if len(parts) == 0 {
		return t.muted_().Render("in sync")
	}
	return strings.Join(parts, "  ")
}

// highlightMatches renders s with the rune positions in matched (indexes
// into s, shifted by offset) rendered in the accent match style.
func highlightMatches(s string, matched []int, offset int, base lipgloss.Style, t theme) string {
	if len(matched) == 0 {
		return base.Render(s)
	}
	set := make(map[int]bool, len(matched))
	for _, i := range matched {
		set[i-offset] = true
	}
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); {
		j := i
		for j < len(runes) && set[j] == set[i] {
			j++
		}
		chunk := string(runes[i:j])
		if set[i] {
			b.WriteString(t.match().Render(chunk))
		} else {
			b.WriteString(base.Render(chunk))
		}
		i = j
	}
	return b.String()
}
