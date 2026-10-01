package project_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/kryft-dev/cdd/internal/project"
)

func newStore(t *testing.T) (*project.Store, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "state", "projects")
	return project.OpenStore(path), path
}

func marks(t *testing.T, s *project.Store) project.Marks {
	t.Helper()

	m, err := s.Marks()
	if err != nil {
		t.Fatalf("Marks: %v", err)
	}
	return m
}

func TestStore_MissingFileIsEmpty(t *testing.T) {
	s, _ := newStore(t)

	m := marks(t, s)
	if len(m.Added()) != 0 || m.IsAdded("/p/a") || m.IsForgotten("/p/a") {
		t.Errorf("a missing file should hold nothing, got added %v", m.Added())
	}
}

func TestStore_AddThenForgetThenReAdd(t *testing.T) {
	s, _ := newStore(t)

	if err := s.Add("/p/a"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if m := marks(t, s); !m.IsAdded("/p/a") || m.IsForgotten("/p/a") {
		t.Errorf("after Add: added=%v forgotten=%v", m.IsAdded("/p/a"), m.IsForgotten("/p/a"))
	}

	if err := s.Forget("/p/a"); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if m := marks(t, s); m.IsAdded("/p/a") || !m.IsForgotten("/p/a") {
		t.Errorf("after Forget: added=%v forgotten=%v", m.IsAdded("/p/a"), m.IsForgotten("/p/a"))
	}

	if err := s.Add("/p/a"); err != nil {
		t.Fatalf("re-Add: %v", err)
	}
	if m := marks(t, s); !m.IsAdded("/p/a") || m.IsForgotten("/p/a") {
		t.Errorf("after re-Add: added=%v forgotten=%v", m.IsAdded("/p/a"), m.IsForgotten("/p/a"))
	}
}

func TestStore_ForgetNeedsNoPriorAdd(t *testing.T) {
	s, _ := newStore(t)

	if err := s.Forget("/p/repo"); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if m := marks(t, s); !m.IsForgotten("/p/repo") || m.IsAdded("/p/repo") {
		t.Errorf("a git Project can be Forgotten without being Added")
	}
}

func TestStore_AddedIsSortedAndSkipsForgotten(t *testing.T) {
	s, _ := newStore(t)
	for _, p := range []string{"/p/b", "/p/c", "/p/a"} {
		if err := s.Add(p); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if err := s.Forget("/p/c"); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	got := marks(t, s).Added()
	if want := []string{"/p/a", "/p/b"}; !slices.Equal(got, want) {
		t.Errorf("Added() = %v, want %v", got, want)
	}
}

func TestStore_WritesTheDocumentedLines(t *testing.T) {
	s, path := newStore(t)
	_ = s.Add("/p/a b")
	_ = s.Forget("/p/c")

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if want := "add /p/a b\nforget /p/c\n"; string(got) != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if !marks(t, s).IsAdded("/p/a b") {
		t.Error("a path holding a space should round-trip")
	}
}

func TestStore_SkipsMalformedLines(t *testing.T) {
	s, path := newStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	junk := "bogus /p/x\nadd relative/path\nadd\n\nforget /p/y\n"
	if err := os.WriteFile(path, []byte(junk), 0o644); err != nil {
		t.Fatal(err)
	}

	m := marks(t, s)
	if len(m.Added()) != 0 || !m.IsForgotten("/p/y") {
		t.Errorf("only the well-formed line should count, got added %v", m.Added())
	}
}

func TestStore_RejectsBadPaths(t *testing.T) {
	s, _ := newStore(t)

	for _, p := range []string{"relative/dir", "/p/new\nline"} {
		if err := s.Add(p); err == nil {
			t.Errorf("Add(%q): want error, got nil", p)
		}
		if err := s.Forget(p); err == nil {
			t.Errorf("Forget(%q): want error, got nil", p)
		}
	}
}

func TestStore_ConcurrentWritesLoseNothing(t *testing.T) {
	s, _ := newStore(t)

	const goroutines = 20
	const perGoroutine = 10
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perGoroutine {
				if err := s.Add("/p/" + strconv.Itoa(g) + "/" + strconv.Itoa(i)); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("Add: %v", err)
	}

	if got, want := len(marks(t, s).Added()), goroutines*perGoroutine; got != want {
		t.Errorf("%d Projects survived, want %d", got, want)
	}
}

func TestStorePath_SitsBesideHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")

	got, err := project.StorePath()
	if err != nil {
		t.Fatalf("StorePath: %v", err)
	}
	if want := filepath.Join("/xdg/data", "cdd", "projects"); got != want {
		t.Errorf("StorePath() = %q, want %q", got, want)
	}
}
