// Package jump resolves the query a Jump starts from into the chosen
// Project's absolute path: it reads History, drops Stale Visits, takes the
// exact-match shortcut, and otherwise runs the Picker.
package jump

import (
	"path/filepath"
	"strings"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/picker"
)

// toRows turns History's latest Visits, already newest first, into Picker
// rows in the same order. counts supplies each Project's Visit total for
// the preview and may be nil. home, when not empty, is shortened to "~" in
// each row's parent directory. toRows is a pure function: it does not touch
// the filesystem or History itself.
func toRows(latest []history.Visit, counts map[string]int, home string) []picker.Row {
	rows := make([]picker.Row, len(latest))
	for i, v := range latest {
		rows[i] = picker.Row{
			Project: picker.Project{
				Kind: shortenHome(filepath.Dir(v.Project), home),
				Name: filepath.Base(v.Project),
				Path: v.Project,
			},
			LastVisit: v.At,
			Visits:    counts[v.Project],
		}
	}
	return rows
}

// shortenHome writes dir with a leading home replaced by "~".
func shortenHome(dir, home string) string {
	if home == "" {
		return dir
	}
	if dir == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(dir, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return dir
}
