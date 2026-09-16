package picker_test

import (
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/picker"
)

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"zero (never visited)", time.Time{}, ""},
		{"seconds", now.Add(-30 * time.Second), "just now"},
		{"minutes", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"one hour", now.Add(-1 * time.Hour), "1 hour ago"},
		{"hours", now.Add(-2 * time.Hour), "2 hours ago"},
		{"yesterday", now.Add(-24 * time.Hour), "yesterday"},
		{"days", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"weeks", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
		{"months", now.Add(-90 * 24 * time.Hour), "3 months ago"},
		{"years", now.Add(-400 * 24 * time.Hour), "1 year ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := picker.RelativeTime(tt.when, now); got != tt.want {
				t.Errorf("RelativeTime(%v) = %q, want %q", tt.when, got, tt.want)
			}
		})
	}
}

func TestRelativeTimeShort(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"zero (never visited)", time.Time{}, ""},
		{"seconds", now.Add(-30 * time.Second), "now"},
		{"minutes", now.Add(-5 * time.Minute), "5m"},
		{"hours", now.Add(-2 * time.Hour), "2h"},
		{"days", now.Add(-3 * 24 * time.Hour), "3d"},
		{"weeks", now.Add(-14 * 24 * time.Hour), "2w"},
		{"months", now.Add(-90 * 24 * time.Hour), "3mo"},
		{"years", now.Add(-400 * 24 * time.Hour), "1y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := picker.RelativeTimeShort(tt.when, now); got != tt.want {
				t.Errorf("RelativeTimeShort(%v) = %q, want %q", tt.when, got, tt.want)
			}
		})
	}
}
