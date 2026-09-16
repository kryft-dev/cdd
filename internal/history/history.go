// Package history stores the ordered record of Visits from which recent
// Projects are derived. It stores Kind and Project only as path strings and
// does not import internal/project.
package history

import (
	"fmt"
	"sort"
	"time"
)

// Source distinguishes how a Visit entered History: a real Jump or a Scan
// seeding.
type Source string

const (
	// SourceJump marks a Visit recorded by an actual Jump.
	SourceJump Source = "jump"
	// SourceScan marks a Visit seeded by a Scan.
	SourceScan Source = "scan"
)

// Visit is a single recorded Jump to a Project, or an entry seeded for a
// Project by a Scan.
type Visit struct {
	// At is the Visit's timestamp, UTC, truncated to second precision.
	At time.Time
	// Source is how the Visit was recorded: SourceJump or SourceScan.
	Source Source
	// Project is the Project's path relative to Root, e.g. "tools/cdd".
	Project string
}

// History is a handle to the on-disk Visit log. It holds no cached state;
// every call reads or writes the file named by path.
type History struct {
	path      string
	maxVisits int
}

// Open opens the History file at path, bounding it to maxVisits Visit
// lines. A missing file is an empty History, not an error; any other read
// failure is returned. maxVisits below 1 is a config error.
func Open(path string, maxVisits int) (*History, error) {
	if maxVisits < 1 {
		return nil, fmt.Errorf("history: max_visits must be at least 1, got %d", maxVisits)
	}
	if _, err := readVisits(path); err != nil {
		return nil, fmt.Errorf("history: open %s: %w", path, err)
	}
	return &History{path: path, maxVisits: maxVisits}, nil
}

// Latest returns each Project's newest Visit, ordered newest-first. Ties on
// the same second break alphabetically by Project. Malformed lines are
// skipped silently.
func (h *History) Latest() ([]Visit, error) {
	visits, err := readVisits(h.path)
	if err != nil {
		return nil, fmt.Errorf("history: latest: %w", err)
	}

	newest := make(map[string]Visit, len(visits))
	for _, v := range visits {
		cur, ok := newest[v.Project]
		if !ok || v.At.After(cur.At) {
			newest[v.Project] = v
		}
	}

	out := make([]Visit, 0, len(newest))
	for _, v := range newest {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At.Equal(out[j].At) {
			return out[i].Project < out[j].Project
		}
		return out[i].At.After(out[j].At)
	})
	return out, nil
}

// Count returns the number of Visit lines recorded for project, including
// Stale Visits, for the Picker's preview. Malformed lines are skipped
// silently.
func (h *History) Count(project string) (int, error) {
	visits, err := readVisits(h.path)
	if err != nil {
		return 0, fmt.Errorf("history: count: %w", err)
	}

	n := 0
	for _, v := range visits {
		if v.Project == project {
			n++
		}
	}
	return n, nil
}
