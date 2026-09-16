package picker_test

import (
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/picker"
)

func TestComputeLayout_PreviewVisibility(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-2 * time.Hour)}

	tests := []struct {
		name        string
		width       int
		wantPreview bool
	}{
		{"wide terminal shows preview", 100, true},
		{"narrow terminal hides preview", 40, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lay := picker.ComputeLayout(8, 1, times, now, tt.width, 24)
			if lay.ShowPreview != tt.wantPreview {
				t.Errorf("ShowPreview = %v, want %v (width %d)", lay.ShowPreview, tt.wantPreview, tt.width)
			}
		})
	}
}

func TestComputeLayout_TimeCompressesBeforeNameTruncates(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	// "2 weeks ago" (11 cols) is long enough to force compression at a
	// tight width, while the Project name should still fit uncompressed.
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	lay := picker.ComputeLayout(12, 1, times, now, 30, 24)

	if !lay.ShortTime {
		t.Fatalf("ShortTime = false, want true at a tight width")
	}
	if lay.NameWidth != 12 {
		t.Errorf("NameWidth = %d, want 12 (unchanged once time compression is enough)", lay.NameWidth)
	}
}

func TestComputeLayout_NameTruncatesToFloorWhenStillTooWide(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	lay := picker.ComputeLayout(40, 1, times, now, 20, 24)

	if lay.NameWidth < 8 {
		t.Errorf("NameWidth = %d, must never go below the floor of 8", lay.NameWidth)
	}
	if lay.NameWidth >= 40 {
		t.Errorf("NameWidth = %d, want it truncated from 40 at a 20-column width", lay.NameWidth)
	}
}

func TestComputeLayout_StatusNeverShrinks(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	lay := picker.ComputeLayout(40, 9, times, now, 15, 24)

	if lay.StatusWidth != 9 {
		t.Errorf("StatusWidth = %d, want 9 (status never shrinks)", lay.StatusWidth)
	}
}

func TestComputeLayout_FooterLines(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		height     int
		wantLegend bool
		wantKeys   bool
	}{
		{"tall terminal shows both", 30, true, true},
		{"under 15 rows drops the legend", 14, false, true},
		{"under 10 rows drops the keys line too", 9, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lay := picker.ComputeLayout(8, 1, nil, now, 100, tt.height)
			if lay.ShowLegend != tt.wantLegend {
				t.Errorf("ShowLegend = %v, want %v", lay.ShowLegend, tt.wantLegend)
			}
			if lay.ShowKeys != tt.wantKeys {
				t.Errorf("ShowKeys = %v, want %v", lay.ShowKeys, tt.wantKeys)
			}
		})
	}
}
