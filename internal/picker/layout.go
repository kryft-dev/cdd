package picker

import "time"

// nameFloor is the smallest a truncated Project name is ever shrunk to.
const nameFloor = 8

// previewFloor is the least remaining width the preview pane needs; below
// it the preview is not drawn at all.
const previewFloor = 30

// caretWidth is the fixed-width caret/gutter column ("   " or " › ").
const caretWidth = 3

// Layout is the Picker's per-frame sizing, recomputed from every
// tea.WindowSizeMsg without losing any Model state.
type Layout struct {
	// NameWidth is the column width for a Project name, after any
	// truncation.
	NameWidth int
	// StatusWidth is the column width for the status glyph cluster. It
	// never shrinks.
	StatusWidth int
	// TimeWidth is the column width for the relative last-visit time.
	TimeWidth int
	// ShortTime reports whether the relative time uses its compressed
	// form (now, 5m, 2h, 3d, 2w, 3mo, 1y).
	ShortTime bool

	// ListWidth is the total width given to the grouped list pane.
	ListWidth int
	// PreviewWidth is the width given to the preview pane, when ShowPreview
	// is true.
	PreviewWidth int
	// ShowPreview reports whether there is room to draw the preview pane.
	ShowPreview bool

	// ListHeight is the number of rows the list body may draw.
	ListHeight int
	// ShowLegend reports whether the footer's glyph legend line is drawn.
	ShowLegend bool
	// ShowKeys reports whether the footer's keys line is drawn.
	ShowKeys bool
}

// ComputeLayout derives a Layout from the terminal size and the content
// that must fit in it: the longest Project name, the widest status glyph
// cluster, and the last-visit times of the currently visible rows.
//
// Degradation order (narrowest terminal wins): the list is sized to its
// natural content width capped at 55% of the terminal; if that leaves under
// previewFloor columns for the preview, the preview is dropped and the list
// takes the full width. Within the list's own budget, relative time
// compresses to its short form first, then names truncate with "…" down to
// nameFloor; the status cluster never shrinks and there is no hard minimum.
func ComputeLayout(longestName, widestStatus int, times []time.Time, now time.Time, width, height int) Layout {
	l := Layout{
		NameWidth:   longestName,
		StatusWidth: widestStatus,
		TimeWidth:   widestTime(times, now, false),
	}

	cap55 := width * 55 / 100
	natural := caretWidth + l.NameWidth + 2 + l.StatusWidth + 2 + l.TimeWidth + 1
	listBudget := min(natural, max(cap55, 1))

	remainder := width - listBudget - 1
	l.ShowPreview = remainder >= previewFloor
	if l.ShowPreview {
		l.PreviewWidth = remainder
	}

	// The width actually available to the list: the full terminal when the
	// preview is not drawn, otherwise the capped budget.
	available := width
	if l.ShowPreview {
		available = listBudget
	}

	if natural > available {
		l.ShortTime = true
		l.TimeWidth = widestTime(times, now, true)
		natural = caretWidth + l.NameWidth + 2 + l.StatusWidth + 2 + l.TimeWidth + 1
	}
	if natural > available {
		overflow := natural - available
		l.NameWidth = max(l.NameWidth-overflow, nameFloor)
	}

	l.ListWidth = available

	l.ShowLegend = height >= 15
	l.ShowKeys = height >= 10
	fixed := 1 // filter line
	if l.ShowLegend {
		fixed++ // rule + legend counted as one reserved line pairing with keys below
	}
	if l.ShowKeys {
		fixed++
	}
	l.ListHeight = max(height-fixed-1, 1) // -1 for the footer rule

	return l
}

// widestTime is the widest rendered relative time among times, using the
// short form when short is true.
func widestTime(times []time.Time, now time.Time, short bool) int {
	w := 0
	for _, t := range times {
		var s string
		if short {
			s = RelativeTimeShort(t, now)
		} else {
			s = RelativeTime(t, now)
		}
		if len(s) > w {
			w = len(s)
		}
	}
	return w
}

// truncateName shortens name to width columns, ending in "…" when it does
// not fit, and never going below nameFloor unless name itself is shorter.
func truncateName(name string, width int) string {
	r := []rune(name)
	if len(r) <= width {
		return name
	}
	if width <= 1 {
		return string(r[:max(width, 0)])
	}
	return string(r[:width-1]) + "…"
}
