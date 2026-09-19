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

// TestComputeLayout_StatusColumnHoldsItsWidthWhileLoading pins the Layout
// computed while every row still reads "…" to the one computed once real
// clusters have landed. Statuses arrive a row at a time with the Picker
// already on screen, and a column sized to the ones that have arrived
// shifted the names, the times and the preview's edge as the rest did.
func TestComputeLayout_StatusColumnHoldsItsWidthWhileLoading(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-2 * time.Hour), now.Add(-3 * 24 * time.Hour)}

	// 1 is the width of the "…" placeholder every row reports before its
	// status lands; 6 is "✓ ? ↑1", a loaded cluster.
	loading := picker.ComputeLayout(20, 1, times, now, 120, 24)
	loaded := picker.ComputeLayout(20, 6, times, now, 120, 24)

	if loading != loaded {
		t.Errorf("layout while loading = %+v, want it identical to the loaded layout %+v", loading, loaded)
	}
}

// TestComputeLayout_StatusColumnGrowsPastTheReserve verifies that a cluster
// wider than the reserve still gets its room: the reserve is a floor, not a
// cap.
func TestComputeLayout_StatusColumnGrowsPastTheReserve(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-2 * time.Hour)}

	lay := picker.ComputeLayout(20, 12, times, now, 120, 24)

	if lay.StatusWidth != 12 {
		t.Errorf("StatusWidth = %d, want 12 (the widest cluster, wider than the reserve)", lay.StatusWidth)
	}
}
