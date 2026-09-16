// Package jump resolves the query a Jump starts from into the chosen
// Project's absolute path: it discovers Projects, orders them by History,
// takes the exact-match shortcut, and otherwise runs the Picker.
package jump

import (
	"sort"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// order applies the ordering rule to projects, given History's latest
// Visits: Projects with a Visit come first, newest Visit first, ties break
// alphabetically by Rel; never-visited Projects follow, alphabetically. A
// Visit for a Project not present in projects (a Stale Visit) contributes
// no row. order is a pure function: it does not touch the filesystem or
// History itself.
func order(projects []project.Project, latest []history.Visit, root string) []picker.Row {
	lastVisit := make(map[string]time.Time, len(latest))
	for _, v := range latest {
		lastVisit[v.Project] = v.At
	}

	ranked := make([]project.Project, len(projects))
	copy(ranked, projects)

	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		aAt, aVisited := lastVisit[a.Rel()]
		bAt, bVisited := lastVisit[b.Rel()]
		if aVisited != bVisited {
			return aVisited
		}
		if aVisited && !aAt.Equal(bAt) {
			return aAt.After(bAt)
		}
		return a.Rel() < b.Rel()
	})

	rows := make([]picker.Row, len(ranked))
	for i, p := range ranked {
		rows[i] = picker.Row{
			Project: picker.Project{
				Kind: p.Kind,
				Name: p.Name,
				Path: p.Abs(root),
			},
			LastVisit: lastVisit[p.Rel()],
		}
	}
	return rows
}
