package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// compact rewrites the History file at path via a temp file and rename,
// keeping the newest maxVisits lines. It drops scan Visits for any Project
// that has a newer jump Visit; malformed lines are already excluded because
// visits comes from a parsed read.
func compact(path string, visits []Visit, maxVisits int) error {
	kept := dropShadowedScans(visits)

	sort.Slice(kept, func(i, j int) bool { return kept[i].At.Before(kept[j].At) })
	if len(kept) > maxVisits {
		kept = kept[len(kept)-maxVisits:]
	}

	return rewrite(path, kept)
}

// dropShadowedScans drops each scan Visit whose Project has a newer jump
// Visit among visits.
func dropShadowedScans(visits []Visit) []Visit {
	latestJump := make(map[string]time.Time, len(visits))
	for _, v := range visits {
		if v.Source != SourceJump {
			continue
		}
		if cur, ok := latestJump[v.Project]; !ok || v.At.After(cur) {
			latestJump[v.Project] = v.At
		}
	}

	kept := make([]Visit, 0, len(visits))
	for _, v := range visits {
		if v.Source == SourceScan {
			if lj, ok := latestJump[v.Project]; ok && lj.After(v.At) {
				continue
			}
		}
		kept = append(kept, v)
	}
	return kept
}

// rewrite writes visits to a temp file beside path and renames it over
// path, so a reader never observes a partially written History file.
func rewrite(path string, visits []Visit) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed

	for _, v := range visits {
		if _, err := tmp.WriteString(formatLine(v)); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("history: compact: %w", err)
	}
	return nil
}
