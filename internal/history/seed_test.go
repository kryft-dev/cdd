package history_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kryft-dev/cdd/internal/history"
)

func TestSeed_AppendsWhenProjectHasNoVisit(t *testing.T) {
	dir := t.TempDir()
	h, err := history.Open(filepath.Join(dir, "history"), 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	at := time.Date(2026, 9, 16, 14, 0, 0, 0, time.UTC)
	if err := h.Seed("tools/cdd", at); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("Latest: got %d visits, want 1", len(latest))
	}
	if latest[0].Source != history.SourceScan {
		t.Errorf("Source = %q, want %q", latest[0].Source, history.SourceScan)
	}
	if !latest[0].At.Equal(at) {
		t.Errorf("At = %v, want %v", latest[0].At, at)
	}
}

func TestSeed_AppendsWhenNewestVisitIsOlderScan(t *testing.T) {
	dir := t.TempDir()
	h, err := history.Open(filepath.Join(dir, "history"), 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	older := time.Date(2026, 9, 16, 14, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 9, 16, 15, 0, 0, 0, time.UTC)

	if err := h.Seed("tools/cdd", older); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}
	if err := h.Seed("tools/cdd", newer); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}

	n, err := h.Count("tools/cdd")
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if n != 2 {
		t.Fatalf("Count = %d, want 2 (newer scan evidence appended)", n)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if !latest[0].At.Equal(newer) {
		t.Errorf("Latest[0].At = %v, want %v", latest[0].At, newer)
	}
}

func TestSeed_SkipsWhenNewestVisitIsScanNotOlder(t *testing.T) {
	dir := t.TempDir()
	h, err := history.Open(filepath.Join(dir, "history"), 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	at := time.Date(2026, 9, 16, 14, 0, 0, 0, time.UTC)
	earlierOrEqual := at // same instant: not strictly older, so no append

	if err := h.Seed("tools/cdd", at); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}
	if err := h.Seed("tools/cdd", earlierOrEqual); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}

	n, err := h.Count("tools/cdd")
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("Count = %d, want 1 (no new evidence, no append)", n)
	}
}

func TestSeed_SkipsWhenNewestVisitIsJump(t *testing.T) {
	dir := t.TempDir()
	h, err := history.Open(filepath.Join(dir, "history"), 1000)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}

	if err := h.Record("tools/cdd"); err != nil {
		t.Fatalf("Record: unexpected error: %v", err)
	}

	// A Scan dated well in the future must still never overwrite a jump
	// Visit.
	future := time.Now().UTC().Add(24 * time.Hour)
	if err := h.Seed("tools/cdd", future); err != nil {
		t.Fatalf("Seed: unexpected error: %v", err)
	}

	n, err := h.Count("tools/cdd")
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("Count = %d, want 1 (jump Visit is never overwritten by Scan)", n)
	}

	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: unexpected error: %v", err)
	}
	if latest[0].Source != history.SourceJump {
		t.Errorf("Source = %q, want %q", latest[0].Source, history.SourceJump)
	}
}
