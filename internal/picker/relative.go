package picker

import (
	"fmt"
	"time"
)

// RelativeTime renders t relative to now for the list row and the preview
// pane's relative column. A zero t (never visited) renders as "".
func RelativeTime(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		n := int(d / time.Minute)
		return plural(n, "minute") + " ago"
	case d < 24*time.Hour:
		n := int(d / time.Hour)
		return plural(n, "hour") + " ago"
	case d < 7*24*time.Hour:
		n := int(d / (24 * time.Hour))
		if n == 1 {
			return "yesterday"
		}
		return plural(n, "day") + " ago"
	case d < 30*24*time.Hour:
		n := int(d / (7 * 24 * time.Hour))
		return plural(n, "week") + " ago"
	case d < 365*24*time.Hour:
		n := int(d / (30 * 24 * time.Hour))
		return plural(n, "month") + " ago"
	default:
		n := int(d / (365 * 24 * time.Hour))
		return plural(n, "year") + " ago"
	}
}

// plural renders "n unit" or "n units".
func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// RelativeTimeShort renders t relative to now in the compressed form narrow
// terminals fall back to: now, 5m, 2h, 3d, 2w, 3mo, 1y. A zero t (never
// visited) renders as "".
func RelativeTimeShort(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(d/(7*24*time.Hour)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d/(30*24*time.Hour)))
	default:
		return fmt.Sprintf("%dy", int(d/(365*24*time.Hour)))
	}
}
