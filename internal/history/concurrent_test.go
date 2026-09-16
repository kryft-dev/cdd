package history_test

import (
	"bufio"
	"os"
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

// TestRecord_ConcurrentAppendsAcrossCompaction is the regression case for
// the lost-Visit bug: a small maxVisits forces compaction to run mid-way
// through the concurrent Records, exercising the append+compact path that
// TestRecord_ConcurrentAppendsLoseNothing's large maxVisits keeps out of the
// way. With withLock correctly serialising every Record (append, and the
// compaction it may trigger) against the others, the outcome is as
// deterministic as running the same 200 Records serially: each compaction
// keeps the newest 100 lines once the count exceeds 110, so 200 Records
// leave exactly 101 lines.
func TestRecord_ConcurrentAppendsAcrossCompaction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	const maxVisits = 100

	h, err := history.Open(path, maxVisits)
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

	const want = 101
	if got := countHistoryLines(t, path); got != want {
		t.Fatalf("line count = %d, want %d (a Visit was lost or a stale compaction ran)", got, want)
	}
}

// countHistoryLines counts the non-empty lines in the History file at path.
func countHistoryLines(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	defer func() { _ = f.Close() }()

	n := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if scanner.Text() != "" {
			n++
		}
	}
	return n
}

func project(g, i int) string {
	return "proj/" + strconv.Itoa(g) + "-" + strconv.Itoa(i)
}

func readAllForTest(t *testing.T, h *history.History) ([]history.Visit, error) {
	t.Helper()
	return h.Latest()
}
