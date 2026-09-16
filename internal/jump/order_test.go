package jump

import (
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// TestOrder_VisitedFirstNewestThenNeverVisited checks the ordering rule:
// Projects with a Visit come first, newest Visit first, ties break
// alphabetically by Rel, and never-visited Projects follow, alphabetically.
func TestOrder_VisitedFirstNewestThenNeverVisited(t *testing.T) {
	projects := []project.Project{
		{Kind: "tools", Name: "cdd"},
		{Kind: "tools", Name: "dotfiles"},
		{Kind: "oss", Name: "lib"},
		{Kind: "work", Name: "api"},
		{Kind: "archive", Name: "old"},
	}

	latest := []history.Visit{
		{Project: "tools/cdd", At: time.Unix(3000, 0)},
		{Project: "work/api", At: time.Unix(5000, 0)},
		{Project: "oss/lib", At: time.Unix(1000, 0)},
	}

	got := order(projects, latest, "/root")

	want := []string{
		"work/api",       // newest Visit
		"tools/cdd",      // next newest Visit
		"oss/lib",        // oldest Visit
		"archive/old",    // never visited, alphabetical
		"tools/dotfiles", // never visited, alphabetical
	}

	assertRowOrder(t, got, want)
}

// TestOrder_TiesBreakAlphabetically checks that Projects sharing the exact
// same Visit timestamp sort alphabetically by Rel among themselves.
func TestOrder_TiesBreakAlphabetically(t *testing.T) {
	projects := []project.Project{
		{Kind: "tools", Name: "zeta"},
		{Kind: "tools", Name: "alpha"},
	}

	tie := time.Unix(1000, 0)
	latest := []history.Visit{
		{Project: "tools/zeta", At: tie},
		{Project: "tools/alpha", At: tie},
	}

	got := order(projects, latest, "/root")

	want := []string{"tools/alpha", "tools/zeta"}
	assertRowOrder(t, got, want)
}

// TestOrder_NeverVisitedAlphabetical checks that, with no Visits at all,
// order falls back to plain alphabetical by Rel.
func TestOrder_NeverVisitedAlphabetical(t *testing.T) {
	projects := []project.Project{
		{Kind: "tools", Name: "zeta"},
		{Kind: "archive", Name: "old"},
		{Kind: "tools", Name: "alpha"},
	}

	got := order(projects, nil, "/root")

	want := []string{"archive/old", "tools/alpha", "tools/zeta"}
	assertRowOrder(t, got, want)
}

// TestOrder_StaleVisitContributesNoRow checks that a Visit for a Project no
// longer discovered (a Stale Visit) does not appear as a row, and that the
// remaining rows order as if it never existed.
func TestOrder_StaleVisitContributesNoRow(t *testing.T) {
	projects := []project.Project{
		{Kind: "tools", Name: "cdd"},
	}

	latest := []history.Visit{
		{Project: "tools/cdd", At: time.Unix(1000, 0)},
		{Project: "gone/vanished", At: time.Unix(9000, 0)},
	}

	got := order(projects, latest, "/root")

	if len(got) != 1 {
		t.Fatalf("order: got %d rows, want 1", len(got))
	}
	if got[0].Project.Kind != "tools" || got[0].Project.Name != "cdd" {
		t.Errorf("row = %+v, want tools/cdd", got[0])
	}
}

// assertRowOrder checks got's rows, in order, have Rel (Kind/Name) matching
// want.
func assertRowOrder(t *testing.T, got []picker.Row, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("order: got %d rows, want %d", len(got), len(want))
	}
	for i, w := range want {
		rel := got[i].Project.Kind + "/" + got[i].Project.Name
		if rel != w {
			t.Errorf("row %d = %q, want %q", i, rel, w)
		}
	}
}
