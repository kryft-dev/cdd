package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover walks each of dirs to any depth and returns the absolute path of
// every Project found, sorted and without duplicates.
//
// The walk stops descending at a Project, so repositories nested inside one
// are not listed, except in a directory named in dirs: that is always
// walked, and listed too when it is itself a Project. Symlinks met during
// the walk are not followed, though a directory in dirs may be one.
// Directories that cannot be read are skipped silently.
//
// Names starting with "." are skipped unless includeHidden is true. Each
// exclude pattern is matched with filepath.Match: a pattern containing "/"
// against a directory's absolute path (e.g. "/home/me/go/pkg/*"), any other
// against its name alone (e.g. "node_modules"). A match skips the directory
// and everything below it.
func Discover(dirs, exclude []string, includeHidden bool) ([]string, error) {
	if err := validatePatterns(exclude); err != nil {
		return nil, err
	}

	w := walker{exclude: exclude, includeHidden: includeHidden, found: map[string]bool{}}
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", dir, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%q is not a directory", dir)
		}
		w.walk(abs, true)
	}

	projects := make([]string, 0, len(w.found))
	for p := range w.found {
		projects = append(projects, p)
	}
	sort.Strings(projects)
	return projects, nil
}

// walker carries one Discover's settings and the Projects found so far.
type walker struct {
	exclude       []string
	includeHidden bool
	found         map[string]bool
}

// walk records dir when it is a Project and descends into its child
// directories. given marks a directory named to Discover, which is walked
// even when it is a Project.
func (w walker) walk(dir string, given bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.Name() == gitEntry {
			w.found[dir] = true
			if !given {
				return
			}
			break
		}
	}

	for _, e := range entries {
		// A symlink's DirEntry is never a directory, so symlinks are not
		// followed.
		if !e.IsDir() || e.Name() == gitEntry {
			continue
		}
		child := filepath.Join(dir, e.Name())
		if w.skip(e.Name(), child) {
			continue
		}
		w.walk(child, false)
	}
}

// skip reports whether the directory named name at path is hidden (and
// hidden directories are not included) or matches an exclude pattern.
func (w walker) skip(name, path string) bool {
	if !w.includeHidden && strings.HasPrefix(name, ".") {
		return true
	}
	for _, p := range w.exclude {
		target := name
		if strings.Contains(p, "/") {
			target = path
		}
		// validatePatterns has already rejected malformed patterns.
		if matched, _ := filepath.Match(p, target); matched {
			return true
		}
	}
	return false
}

// validatePatterns returns an error if any exclude pattern is malformed,
// per filepath.Match.
func validatePatterns(exclude []string) error {
	for _, p := range exclude {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}
	return nil
}
