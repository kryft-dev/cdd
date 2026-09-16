package main

// PROTOTYPE: adaptive palette and status glyph rendering for the Picker mock.

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// theme holds the few colours the Picker needs, chosen per terminal
// background via lipgloss.LightDark. Body text keeps the terminal's own
// foreground so it adapts for free.
type theme struct {
	dark                                          bool
	muted, green, yellow, blue, red, purple, rule color.Color
	selBg, selFg, barBg, barFg                    color.Color
}

func newTheme(dark bool) theme {
	ld := lipgloss.LightDark(dark)
	c := lipgloss.Color
	return theme{
		dark:   dark,
		muted:  ld(c("#6E7781"), c("#8B949E")),
		green:  ld(c("#1A7F37"), c("#3FB950")),
		yellow: ld(c("#9A6700"), c("#D29922")),
		blue:   ld(c("#0969DA"), c("#58A6FF")),
		red:    ld(c("#CF222E"), c("#F85149")),
		purple: ld(c("#8250DF"), c("#A371F7")),
		rule:   ld(c("#D0D7DE"), c("#30363D")),
		selBg:  ld(c("#DDF4FF"), c("#1F3552")),
		selFg:  ld(c("#0550AE"), c("#CAE8FF")),
		barBg:  c("#FF5FAF"),
		barFg:  c("#000000"),
	}
}

func (t theme) S() lipgloss.Style               { return lipgloss.NewStyle() }
func (t theme) fg(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
func (t theme) Muted() lipgloss.Style           { return t.fg(t.muted) }
func (t theme) Header() lipgloss.Style          { return t.fg(t.muted).Bold(true) }
func (t theme) Rule(w int) string               { return t.fg(t.rule).Render(strings.Repeat("─", w)) }
func (t theme) Match(base lipgloss.Style) lipgloss.Style {
	return base.Foreground(t.purple).Bold(true).Underline(true)
}

// glyphs: the status vocabulary. Same glyphs in every variant so the reaction
// is about layout, not icons.
const (
	gClean     = "✓"
	gDirty     = "●"
	gUntracked = "?"
	gAhead     = "↑"
	gBehind    = "↓"
	gTimeout   = "!"
	gNoRepo    = "—"
	gPending   = "…"
)

// statusCell renders the compact coloured glyph cluster for one row, e.g.
// "✓", "● ?", "● ↑3↓2". base carries the row background when selected.
func (t theme) statusCell(p project, base lipgloss.Style) string {
	f := func(c color.Color, s string) string { return base.Foreground(c).Render(s) }
	switch {
	case !p.Repo:
		return f(t.muted, gNoRepo)
	case !p.Loaded:
		return f(t.muted, gPending)
	case p.Timeout:
		return f(t.red, gTimeout)
	}
	var parts []string
	if p.Dirty {
		parts = append(parts, f(t.yellow, gDirty))
	} else {
		parts = append(parts, f(t.green, gClean))
	}
	if p.Untracked {
		parts = append(parts, f(t.blue, gUntracked))
	}
	sync := ""
	if p.Ahead > 0 {
		sync += gAhead + strconv.Itoa(p.Ahead)
	}
	if p.Behind > 0 {
		sync += gBehind + strconv.Itoa(p.Behind)
	}
	if sync != "" {
		parts = append(parts, f(t.purple, sync))
	}
	return strings.Join(parts, base.Render(" "))
}

// statusWidth is the plain width of statusCell, for column alignment.
func statusWidth(p project) int {
	switch {
	case !p.Repo, !p.Loaded, p.Timeout:
		return 1
	}
	w := 1
	if p.Untracked {
		w += 2
	}
	sync := 0
	if p.Ahead > 0 {
		sync += 1 + len(strconv.Itoa(p.Ahead))
	}
	if p.Behind > 0 {
		sync += 1 + len(strconv.Itoa(p.Behind))
	}
	if sync > 0 {
		w += 1 + sync
	}
	return w
}

// statusWords spells the status out, for variants that skip the legend.
func (t theme) statusWords(p project, base lipgloss.Style) string {
	f := func(c color.Color, s string) string { return base.Foreground(c).Render(s) }
	switch {
	case !p.Repo:
		return f(t.muted, gNoRepo+" not a repo")
	case !p.Loaded:
		return f(t.muted, gPending+" loading")
	case p.Timeout:
		return f(t.red, gTimeout+" git timed out")
	}
	var parts []string
	if p.Dirty {
		parts = append(parts, f(t.yellow, gDirty+" modified"))
	} else {
		parts = append(parts, f(t.green, gClean+" clean"))
	}
	if p.Untracked {
		parts = append(parts, f(t.blue, gUntracked+" untracked"))
	}
	if p.Ahead > 0 {
		parts = append(parts, f(t.purple, gAhead+strconv.Itoa(p.Ahead)+" ahead"))
	}
	if p.Behind > 0 {
		parts = append(parts, f(t.purple, gBehind+strconv.Itoa(p.Behind)+" behind"))
	}
	if !p.Upstream && p.Loaded && p.Repo {
		parts = append(parts, f(t.muted, "no upstream"))
	}
	return strings.Join(parts, base.Render("  "))
}

// legend is the plain-word footer key for the glyphs.
func (t theme) legend() string {
	f := func(c color.Color, g, w string) string { return t.fg(c).Render(g) + t.Muted().Render(" "+w) }
	return strings.Join([]string{
		f(t.green, gClean, "clean"),
		f(t.yellow, gDirty, "modified"),
		f(t.blue, gUntracked, "untracked"),
		f(t.purple, gAhead, "ahead"),
		f(t.purple, gBehind, "behind"),
		f(t.red, gTimeout, "unknown"),
		f(t.muted, gNoRepo, "not a repo"),
		f(t.muted, gPending, "loading"),
	}, "  ")
}

func (t theme) keys(items ...string) string {
	var b []string
	for i := 0; i+1 < len(items); i += 2 {
		b = append(b, t.fg(t.muted).Bold(true).Render(items[i])+t.Muted().Render(" "+items[i+1]))
	}
	return strings.Join(b, t.Muted().Render(" · "))
}

// highlight renders s with the runes at matched indexes emphasised. matched
// are indexes into s (rune positions); offset shifts them for substrings.
func highlight(s string, matched []int, offset int, base, hl lipgloss.Style) string {
	set := map[int]bool{}
	for _, i := range matched {
		set[i-offset] = true
	}
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); {
		j := i
		for j < len(runes) && set[j] == set[i] {
			j++
		}
		chunk := string(runes[i:j])
		if set[i] {
			b.WriteString(hl.Render(chunk))
		} else {
			b.WriteString(base.Render(chunk))
		}
		i = j
	}
	return b.String()
}
