package history_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
)

// seedLines writes n synthetic, strictly increasing jump lines directly to
// path, one per distinct Project, bypassing the package so the test
// controls timestamps precisely.
func seedLines(t *testing.T, path string, n int) {
	t.Helper()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var sb strings.Builder
	for i := range n {
		at := base.Add(time.Duration(i) * time.Second)
		fmt.Fprintf(&sb, "%s\tjump\tproj/%d\n", at.Format(time.RFC3339), i)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("WriteFile: unexpected error: %v", err)
	}
}

func countLines(t *testing.T, path string) int {
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

func TestRecord_CompactsPastTenPercentThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	maxVisits := 10

	// 10 lines plus one append is 11, not yet more than 10% over 10 (the
	// threshold is exceeded only once the count passes 11).
	seedLines(t, path, 10)

	h, err := history.Open(path, maxVisits)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	if err := h.Record("proj/new"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}
	if got := countLines(t, path); got != 11 {
		t.Fatalf("line count = %d, want 11 (no compaction yet)", got)
	}

	// One more append pushes the file past the threshold and must compact
	// down to maxVisits lines.
	if err := h.Record("proj/newer"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}
	if got := countLines(t, path); got != maxVisits {
		t.Fatalf("line count = %d, want %d after compaction", got, maxVisits)
	}
}

func TestCompact_KeepsNewestLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	maxVisits := 5

	seedLines(t, path, 20) // proj/0 .. proj/19, oldest to newest

	h, err := history.Open(path, maxVisits)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	if err := h.Record("proj/last"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != maxVisits {
		t.Fatalf("Latest: got %d visits, want %d", len(latest), maxVisits)
	}
	if latest[0].Project != "proj/last" {
		t.Errorf("Latest[0].Project = %q, want %q (most recent survives compaction)", latest[0].Project, "proj/last")
	}
	for _, v := range latest {
		if v.Project == "proj/0" {
			t.Errorf("Latest: found proj/0, oldest Visit should have been compacted away")
		}
	}
}

func TestCompact_DropsScanShadowedByNewerJump(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	maxVisits := 3

	content := "2026-01-01T00:00:00Z\tscan\tproj/a\n" +
		"2026-01-01T00:00:01Z\tjump\tproj/a\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: unexpected error: %v", err)
	}

	h, err := history.Open(path, maxVisits)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	// Force compaction with enough appends to cross the threshold.
	if err := h.Record("proj/b"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}
	if err := h.Record("proj/c"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}
	if err := h.Record("proj/d"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "\tscan\tproj/a") {
			t.Errorf("compacted file still contains shadowed scan line: %q", scanner.Text())
		}
	}
}

func TestOpen_MalformedLinesDoNotPreventFutureCompaction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")

	content := "garbage line with no tabs\n" +
		"2026-01-01T00:00:00Z\tjump\tproj/a\n" +
		"2026-01-01T00:00:01\tjump\tproj/b\n" // malformed timestamp
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
	if len(latest) != 1 {
		t.Fatalf("Latest: got %d visits, want 1 (malformed lines skipped)", len(latest))
	}
	if latest[0].Project != "proj/a" {
		t.Errorf("Project = %q, want %q", latest[0].Project, "proj/a")
	}
}
