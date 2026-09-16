package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Discover walks root, treating every directory directly under root as a
// Kind and every directory directly under a Kind as a Project. Names
// starting with "." are skipped at both levels unless includeHidden is
// true. Each entry of exclude is matched with path.Match, once against the
// Kind name alone (e.g. "archive") and once against the Project's Rel
// (e.g. "tools/scratch", "archive/*", "*/node_modules"); a match at either
// level skips that Kind or Project. Symlinked directories count as
// directories; non-directories are ignored. The result is sorted by Rel.
func Discover(root string, exclude []string, includeHidden bool) ([]Project, error) {
	if err := validatePatterns(exclude); err != nil {
		return nil, err
	}

	kindEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read root %q: %w", root, err)
	}

	var projects []Project
	for _, kindEntry := range kindEntries {
		kind := kindEntry.Name()
		if skipHidden(kind, includeHidden) {
			continue
		}

		isDir, err := isDirEntry(root, kindEntry)
		if err != nil {
			return nil, err
		}
		if !isDir {
			continue
		}

		matched, err := matchAny(exclude, kind)
		if err != nil {
			return nil, err
		}
		if matched {
			continue
		}

		kindPath := filepath.Join(root, kind)
		projEntries, err := os.ReadDir(kindPath)
		if err != nil {
			return nil, fmt.Errorf("read kind %q: %w", kindPath, err)
		}

		for _, projEntry := range projEntries {
			name := projEntry.Name()
			if skipHidden(name, includeHidden) {
				continue
			}

			isDir, err := isDirEntry(kindPath, projEntry)
			if err != nil {
				return nil, err
			}
			if !isDir {
				continue
			}

			p := Project{Kind: kind, Name: name}

			matched, err := matchAny(exclude, p.Rel())
			if err != nil {
				return nil, err
			}
			if matched {
				continue
			}

			projects = append(projects, p)
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Rel() < projects[j].Rel()
	})

	return projects, nil
}

// skipHidden reports whether name should be skipped for starting with "."
// when includeHidden is false.
func skipHidden(name string, includeHidden bool) bool {
	return !includeHidden && strings.HasPrefix(name, ".")
}

// isDirEntry reports whether entry, found in dir, is a directory. A
// symlink is resolved and counts as a directory when its target is one; a
// broken symlink is treated as not a directory.
func isDirEntry(dir string, entry os.DirEntry) (bool, error) {
	if entry.Type()&os.ModeSymlink == 0 {
		return entry.IsDir(), nil
	}

	info, err := os.Stat(filepath.Join(dir, entry.Name()))
	if err != nil {
		// A broken symlink is not a directory.
		return false, nil
	}
	return info.IsDir(), nil
}

// validatePatterns returns an error if any exclude pattern is malformed,
// per path.Match.
func validatePatterns(exclude []string) error {
	for _, p := range exclude {
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}
	return nil
}

// matchAny reports whether s matches any of the exclude patterns.
func matchAny(exclude []string, s string) (bool, error) {
	for _, p := range exclude {
		matched, err := path.Match(p, s)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
