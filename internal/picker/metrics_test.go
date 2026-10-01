package picker_test

import (
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/picker"
)

func TestComputeMetrics_PreviewVisibility(t *testing.T) {
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
			met := picker.ComputeMetrics(8, 1, times, now, tt.width, 24)
			if met.ShowPreview != tt.wantPreview {
				t.Errorf("ShowPreview = %v, want %v (width %d)", met.ShowPreview, tt.wantPreview, tt.width)
			}
		})
	}
}

func TestComputeMetrics_TimeCompressesBeforeNameTruncates(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	// "2 weeks ago" (11 cols) is long enough to force compression at a
	// tight width, while the Project name should still fit uncompressed.
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	met := picker.ComputeMetrics(12, 1, times, now, 30, 24)

	if !met.ShortTime {
		t.Fatalf("ShortTime = false, want true at a tight width")
	}
	if met.NameWidth != 12 {
		t.Errorf("NameWidth = %d, want 12 (unchanged once time compression is enough)", met.NameWidth)
	}
}

func TestComputeMetrics_NameTruncatesToFloorWhenStillTooWide(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	met := picker.ComputeMetrics(40, 1, times, now, 20, 24)

	if met.NameWidth < 8 {
		t.Errorf("NameWidth = %d, must never go below the floor of 8", met.NameWidth)
	}
	if met.NameWidth >= 40 {
		t.Errorf("NameWidth = %d, want it truncated from 40 at a 20-column width", met.NameWidth)
	}
}

func TestComputeMetrics_StatusNeverShrinks(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-14 * 24 * time.Hour)}

	met := picker.ComputeMetrics(40, 9, times, now, 15, 24)

	if met.StatusWidth != 9 {
		t.Errorf("StatusWidth = %d, want 9 (status never shrinks)", met.StatusWidth)
	}
}

func TestComputeMetrics_FooterLines(t *testing.T) {
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
			met := picker.ComputeMetrics(8, 1, nil, now, 100, tt.height)
			if met.ShowLegend != tt.wantLegend {
				t.Errorf("ShowLegend = %v, want %v", met.ShowLegend, tt.wantLegend)
			}
			if met.ShowKeys != tt.wantKeys {
				t.Errorf("ShowKeys = %v, want %v", met.ShowKeys, tt.wantKeys)
			}
		})
	}
}

// TestComputeMetrics_StatusColumnHoldsItsWidthWhileLoading pins the Metrics
// computed while every row still reads "…" to the one computed once real
// clusters have landed. Statuses arrive a row at a time with the Picker
// already on screen, and a column sized to the ones that have arrived
// shifted the names, the times and the preview's edge as the rest did.
func TestComputeMetrics_StatusColumnHoldsItsWidthWhileLoading(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-2 * time.Hour), now.Add(-3 * 24 * time.Hour)}

	// 1 is the width of the "…" placeholder every row reports before its
	// status lands; 6 is "✓ ? ↑1", a loaded cluster.
	loading := picker.ComputeMetrics(20, 1, times, now, 120, 24)
	loaded := picker.ComputeMetrics(20, 6, times, now, 120, 24)

	if loading != loaded {
		t.Errorf("metrics while loading = %+v, want it identical to the loaded metrics %+v", loading, loaded)
	}
}

// TestComputeMetrics_StatusColumnGrowsPastTheReserve verifies that a cluster
// wider than the reserve still gets its room: the reserve is a floor, not a
// cap.
func TestComputeMetrics_StatusColumnGrowsPastTheReserve(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	times := []time.Time{now.Add(-2 * time.Hour)}

	met := picker.ComputeMetrics(20, 12, times, now, 120, 24)

	if met.StatusWidth != 12 {
		t.Errorf("StatusWidth = %d, want 12 (the widest cluster, wider than the reserve)", met.StatusWidth)
	}
}
