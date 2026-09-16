package history

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// maxLineBytes bounds a single appended Visit line, comfortably under the
// 4 KiB advisory limit.
const maxLineBytes = 4096

// Record appends a jump Visit for project, timestamped now (UTC, second
// precision).
func (h *History) Record(project string) error {
	v := Visit{At: time.Now().UTC(), Source: SourceJump, Project: project}
	return h.withLock(func(f *os.File) error {
		return h.appendAndCompact(f, v)
	})
}

// Seed appends a scan Visit for project, timestamped at, but only if the
// Project has no Visit at all, or its newest Visit is a scan line older
// than at. A jump Visit is never overwritten by a Scan.
func (h *History) Seed(project string, at time.Time) error {
	v := Visit{At: at.UTC(), Source: SourceScan, Project: project}
	return h.withLock(func(f *os.File) error {
		visits, err := parseVisits(f)
		if err != nil {
			return err
		}

		newest, found := newestFor(visits, project)
		if found && !(newest.Source == SourceScan && newest.At.Before(v.At)) {
			return nil
		}

		return h.appendAndCompact(f, v)
	})
}

// newestFor returns the newest Visit recorded for project, if any.
func newestFor(visits []Visit, project string) (Visit, bool) {
	var newest Visit
	found := false
	for _, v := range visits {
		if v.Project != project {
			continue
		}
		if !found || v.At.After(newest.At) {
			newest = v
			found = true
		}
	}
	return newest, found
}

// appendAndCompact appends v to the already-locked, already-open History
// file f, then compacts if the line count now exceeds maxVisits by more
// than 10 percent.
func (h *History) appendAndCompact(f *os.File, v Visit) error {
	line := formatLine(v)
	if len(line) > maxLineBytes {
		return fmt.Errorf("history: line for project %q exceeds %d bytes", v.Project, maxLineBytes)
	}

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	if _, err := f.WriteString(line); err != nil {
		return err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	visits, err := parseVisits(f)
	if err != nil {
		return err
	}

	if !exceedsThreshold(len(visits), h.maxVisits) {
		return nil
	}
	return compact(h.path, visits, h.maxVisits)
}

// exceedsThreshold reports whether total exceeds maxVisits by more than 10
// percent.
func exceedsThreshold(total, maxVisits int) bool {
	return total*10 > maxVisits*11
}

// withLock opens the History file for read/write (creating it if absent),
// takes an advisory exclusive flock, runs fn, then unlocks and closes.
func (h *History) withLock(fn func(f *os.File) error) error {
	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return fmt.Errorf("history: %w", err)
	}

	f, err := os.OpenFile(h.path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("history: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("history: lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	if err := fn(f); err != nil {
		return fmt.Errorf("history: %w", err)
	}
	return nil
}
