package history_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
)

func TestOpen_MissingFileIsEmptyHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 0 {
		t.Fatalf("Latest: got %d visits, want 0", len(latest))
	}
}

func TestOpen_RejectsMaxVisitsBelowOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	if _, err := history.Open(path, 0); err == nil {
		t.Fatal("Open: want error for max_visits below 1, got nil")
	}
}

func TestRecord_AppendsAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	if err := h.Record("tools/cdd"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("Latest: got %d visits, want 1", len(latest))
	}
	if latest[0].Project != "tools/cdd" {
		t.Errorf("Project = %q, want %q", latest[0].Project, "tools/cdd")
	}
	if latest[0].Source != history.SourceJump {
		t.Errorf("Source = %q, want %q", latest[0].Source, history.SourceJump)
	}
	if latest[0].At.After(time.Now().UTC()) {
		t.Errorf("At = %v, want not after now", latest[0].At)
	}
}

func TestLatest_NewestPerProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	// tools/other's newest line predates tools/cdd's newest line, so Latest
	// must order tools/cdd first despite tools/other appearing earlier in
	// the file too.
	content := "2026-09-16T14:00:00Z\tjump\ttools/other\n" +
		"2026-09-16T14:01:00Z\tjump\ttools/cdd\n" +
		"2026-09-16T14:02:00Z\tjump\ttools/other\n" +
		"2026-09-16T14:03:00Z\tjump\ttools/cdd\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: unexpected error: %v", err)
	}

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("Latest: got %d visits, want 2 (one per Project)", len(latest))
	}
	if latest[0].Project != "tools/cdd" {
		t.Errorf("Latest[0].Project = %q, want %q (most recently jumped)", latest[0].Project, "tools/cdd")
	}
	if latest[1].Project != "tools/other" {
		t.Errorf("Latest[1].Project = %q, want %q", latest[1].Project, "tools/other")
	}
}

func TestCount_PerProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	for range 3 {
		if err := h.Record("tools/cdd"); err != nil {
			t.Fatalf("Record: unexpected error: %v", err)
		}
	}
	if err := h.Record("tools/other"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	n, err := h.Count("tools/cdd")
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("Count(tools/cdd) = %d, want 3", n)
	}

	n, err = h.Count("tools/never-visited")
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("Count(tools/never-visited) = %d, want 0", n)
	}
}

func TestCounts_AllProjectsInOneRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	for range 3 {
		if err := h.Record("tools/cdd"); err != nil {
			t.Fatalf("Record: unexpected error: %v", err)
		}
	}
	if err := h.Record("tools/other"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	counts, err := h.Counts()
	if err != nil {
		t.Fatalf("Counts: unexpected error: %v", err)
	}
	want := map[string]int{"tools/cdd": 3, "tools/other": 1}
	if len(counts) != len(want) {
		t.Fatalf("Counts: got %d Projects, want %d", len(counts), len(want))
	}
	for project, n := range want {
		if counts[project] != n {
			t.Errorf("Counts[%q] = %d, want %d", project, counts[project], n)
		}
	}
	if counts["never/visited"] != 0 {
		t.Errorf("Counts of an unvisited Project = %d, want 0", counts["never/visited"])
	}
}

func TestReadVisits_SkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	content := "2026-09-16T14:03:22Z\tjump\ttools/cdd\n" +
		"not a valid line\n" +
		"2026-09-16T14:04:00Z\tflyby\ttools/bad-source\n" +
		"2026-09-16T14:05:00Z\tjump\ttools/good\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: unexpected error: %v", err)
	}

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("Latest: got %d visits, want 2 (malformed lines skipped)", len(latest))
	}
}

// TestReadVisits_SkipsOverLongMalformedLine is the regression case for a
// corrupt line beyond bufio.Scanner's default 64 KiB token limit: it must
// be skipped as malformed like any other, not abort the read for the whole
// History file.
func TestReadVisits_SkipsOverLongMalformedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	garbage := strings.Repeat("x", 100*1024) // well past the 64 KiB limit
	content := "2026-09-16T14:03:22Z\tjump\ttools/cdd\n" +
		garbage + "\n" +
		"2026-09-16T14:05:00Z\tjump\ttools/good\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: unexpected error: %v", err)
	}

	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("Latest: got %d visits, want 2 (over-long line skipped, surrounding lines parsed)", len(latest))
	}
}
