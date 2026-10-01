package project

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kryft-dev/cdd/internal/history"
)

// maxLineBytes bounds one appended line, comfortably under the 4 KiB
// advisory limit below which a single O_APPEND write is not interleaved
// with another process's.
const maxLineBytes = 4096

const (
	verbAdd    = "add"
	verbForget = "forget"
)

// Store is a handle to the on-disk file recording the Projects the user has
// Added or Forgotten by hand, one "add <abs path>" or "forget <abs path>"
// line each. The last line for a path wins. It holds no cached state; every
// call reads or writes the file named by path. cdd owns this file; it never
// writes into a Project.
type Store struct {
	path string
}

// StorePath returns the default Store file path, "projects" in the
// directory holding History.
func StorePath() (string, error) {
	hist, err := history.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(hist), "projects"), nil
}

// OpenStore returns a Store over the file at path. A missing file is an
// empty Store, not an error; it is created by the first Add or Forget.
func OpenStore(path string) *Store {
	return &Store{path: path}
}

// Add marks project, an absolute path, as Added, bringing back a Forgotten
// one. It does not check that the directory exists; the cli does.
func (s *Store) Add(project string) error {
	return s.append(verbAdd, project)
}

// Forget marks project, an absolute path, as Forgotten: hidden from the
// Picker and skipped by a Scan until it is Added again.
func (s *Store) Forget(project string) error {
	return s.append(verbForget, project)
}

// append writes one line with a single O_APPEND write, which the kernel
// does not interleave with another process's, so concurrent writers lose
// nothing and need no lock: the file is never rewritten.
func (s *Store) append(verb, project string) error {
	if !filepath.IsAbs(project) {
		return fmt.Errorf("project: %q is not an absolute path", project)
	}
	if strings.Contains(project, "\n") {
		return fmt.Errorf("project: %q contains a newline", project)
	}
	line := verb + " " + project + "\n"
	if len(line) > maxLineBytes {
		return fmt.Errorf("project: line for %q exceeds %d bytes", project, maxLineBytes)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("project: %w", err)
	}
	f, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("project: %w", err)
	}
	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return fmt.Errorf("project: %w", err)
	}
	return f.Close()
}

// Marks is what the Store holds once the last line for each path has won.
type Marks struct {
	added     map[string]bool
	forgotten map[string]bool
}

// Marks reads the Store. Malformed lines are skipped silently.
func (s *Store) Marks() (Marks, error) {
	m := Marks{added: map[string]bool{}, forgotten: map[string]bool{}}
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return Marks{}, fmt.Errorf("project: %w", err)
	}
	defer func() { _ = f.Close() }()

	// bufio.Reader, not Scanner, so one over-long garbage line is skipped
	// like any malformed line instead of aborting the read.
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadString('\n')
		if verb, path, ok := parseLine(strings.TrimSuffix(line, "\n")); ok {
			m.added[path] = verb == verbAdd
			m.forgotten[path] = verb == verbForget
		}
		if err == io.EOF {
			return m, nil
		}
		if err != nil {
			return Marks{}, fmt.Errorf("project: %w", err)
		}
	}
}

// parseLine splits a line into its verb and absolute path, reporting
// ok=false for anything else.
func parseLine(line string) (verb, path string, ok bool) {
	verb, path, found := strings.Cut(line, " ")
	if !found || (verb != verbAdd && verb != verbForget) || !filepath.IsAbs(path) {
		return "", "", false
	}
	return verb, path, true
}

// IsAdded reports whether project's last line is an add.
func (m Marks) IsAdded(project string) bool { return m.added[project] }

// IsForgotten reports whether project's last line is a forget.
func (m Marks) IsForgotten(project string) bool { return m.forgotten[project] }

// Added returns every Added path, sorted.
func (m Marks) Added() []string {
	out := make([]string, 0, len(m.added))
	for p, ok := range m.added {
		if ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
