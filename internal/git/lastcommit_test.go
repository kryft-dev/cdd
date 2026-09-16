package git_test

import (
	"context"
	"testing"
	"time"

	cdgit "github.com/kryft-dev/cdd/internal/git"
)

func TestLastCommit(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	commit(t, dir, "initial")

	before := time.Now().Add(-time.Minute)

	got, ok, err := cdgit.LastCommit(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got.Before(before) || got.After(time.Now().Add(time.Minute)) {
		t.Errorf("LastCommit = %v, want close to now", got)
	}
}

func TestLastCommit_Unborn(t *testing.T) {
	dir := newRepo(t)

	_, ok, err := cdgit.LastCommit(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}
	if ok {
		t.Errorf("ok = true, want false for an unborn branch")
	}
}

func TestLastCommit_NotRepo(t *testing.T) {
	dir := t.TempDir()

	_, ok, err := cdgit.LastCommit(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}
	if ok {
		t.Errorf("ok = true, want false for a non-repository directory")
	}
}
