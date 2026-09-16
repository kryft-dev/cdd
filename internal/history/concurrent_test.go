package history_test

import (
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/kryft-dev/cdd/internal/history"
)

func TestRecord_ConcurrentAppendsLoseNothing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	// A large max_visits keeps compaction out of the way so this test
	// isolates the flock append path.
	h, err := history.Open(path, 100000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	const goroutines = 20
	const perGoroutine = 10

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range perGoroutine {
				if err := h.Record(project(g, i)); err != nil {
					errCh <- err
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	visits, err := readAllForTest(t, h)
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	want := goroutines * perGoroutine
	if len(visits) != want {
		t.Fatalf("recorded %d distinct Projects, want %d (some Visits were lost)", len(visits), want)
	}
}

func project(g, i int) string {
	return "proj/" + strconv.Itoa(g) + "-" + strconv.Itoa(i)
}

func readAllForTest(t *testing.T, h *history.History) ([]history.Visit, error) {
	t.Helper()
	return h.Latest()
}
