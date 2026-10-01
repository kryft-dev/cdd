package jump

import (
	"os"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/project"
)

// listed returns the Visits the Picker shows: latest without its Forgotten
// and Stale Visits, then one Visit-less entry for each Added Project that
// has no Visit, in path order. A Visit is Stale when its Project holds no
// git repository and is not an Added directory that still exists.
func listed(latest []history.Visit, marks project.Marks) []history.Visit {
	out := make([]history.Visit, 0, len(latest))
	seen := make(map[string]bool, len(latest))
	for _, v := range latest {
		seen[v.Project] = true
		if live(v.Project, marks) {
			out = append(out, v)
		}
	}
	for _, p := range marks.Added() {
		if !seen[p] && isDir(p) {
			out = append(out, history.Visit{Project: p})
		}
	}
	return out
}

// live reports whether path is still a Project: not Forgotten, and either a
// git repository or an Added directory that exists.
func live(path string, marks project.Marks) bool {
	if marks.IsForgotten(path) {
		return false
	}
	return project.IsRepo(path) || (marks.IsAdded(path) && isDir(path))
}

// isDir reports whether path is an existing directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
