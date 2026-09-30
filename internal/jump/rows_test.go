package jump

import (
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
)

// TestToRows_KeepsHistoryOrder checks that rows come out in the order
// History's latest Visits are given, newest first, each carrying its
// Visit's time.
func TestToRows_KeepsHistoryOrder(t *testing.T) {
	latest := []history.Visit{
		{Project: "/home/me/work/api", At: time.Unix(5000, 0)},
		{Project: "/home/me/tools/cdd", At: time.Unix(3000, 0)},
		{Project: "/srv/lib", At: time.Unix(1000, 0)},
	}

	got := toRows(latest, nil, "/home/me")

	if len(got) != len(latest) {
		t.Fatalf("toRows: got %d rows, want %d", len(got), len(latest))
	}
	for i, v := range latest {
		if got[i].Project.Path != v.Project || !got[i].LastVisit.Equal(v.At) {
			t.Errorf("row %d = %+v, want %s at %v", i, got[i], v.Project, v.At)
		}
	}
}

// TestToRows_SplitsParentAndName checks each row's parent directory, with
// the home directory shortened to "~", and its name.
func TestToRows_SplitsParentAndName(t *testing.T) {
	tests := []struct {
		path, home, dir, name string
	}{
		{"/home/me/tools/cdd", "/home/me", "~/tools", "cdd"},
		{"/home/me/cdd", "/home/me", "~", "cdd"},
		{"/home/meta/cdd", "/home/me", "/home/meta", "cdd"},
		{"/srv/lib", "/home/me", "/srv", "lib"},
		{"/home/me/tools/cdd", "", "/home/me/tools", "cdd"},
	}

	for _, tt := range tests {
		got := toRows([]history.Visit{{Project: tt.path}}, nil, tt.home)[0].Project
		if got.Kind != tt.dir || got.Name != tt.name {
			t.Errorf("toRows(%q, home %q) = %q + %q, want %q + %q", tt.path, tt.home, got.Kind, got.Name, tt.dir, tt.name)
		}
	}
}

// TestToRows_FillsVisitCounts checks that each row carries the Project's
// Visit total from counts, and zero for a Project counts does not know.
func TestToRows_FillsVisitCounts(t *testing.T) {
	latest := []history.Visit{{Project: "/p/cdd"}, {Project: "/p/dotfiles"}}
	counts := map[string]int{"/p/cdd": 4}

	got := toRows(latest, counts, "")

	if got[0].Visits != 4 {
		t.Errorf("/p/cdd Visits = %d, want 4", got[0].Visits)
	}
	if got[1].Visits != 0 {
		t.Errorf("/p/dotfiles Visits = %d, want 0", got[1].Visits)
	}
}
