package git_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kryft-dev/cdd/internal/git"
)

func TestRemoteURL_PrefersOrigin(t *testing.T) {
	dir := newRepo(t)
	run(t, dir, "remote", "add", "a-first", "git@example.com:other/repo.git")
	run(t, dir, "remote", "add", "origin", "git@github.com:owner/repo.git")

	got, err := git.RemoteURL(context.Background(), dir)
	if err != nil || got != "https://github.com/owner/repo" {
		t.Errorf("RemoteURL = %q, %v", got, err)
	}
}

func TestRemoteURL_FallsBackToTheFirstRemote(t *testing.T) {
	dir := newRepo(t)
	run(t, dir, "remote", "add", "upstream", "https://gitlab.com/owner/group/repo.git")
	run(t, dir, "remote", "add", "fork", "https://gitlab.com/me/repo.git")

	got, err := git.RemoteURL(context.Background(), dir)
	if err != nil || got != "https://gitlab.com/me/repo" {
		t.Errorf("RemoteURL = %q, %v", got, err)
	}
}

func TestRemoteURL_NoRemote(t *testing.T) {
	_, err := git.RemoteURL(context.Background(), newRepo(t))
	if !errors.Is(err, git.ErrNoRemote) {
		t.Errorf("err = %v, want ErrNoRemote", err)
	}
}

func TestRemoteURL_NotARepo(t *testing.T) {
	_, err := git.RemoteURL(context.Background(), t.TempDir())
	if !errors.Is(err, git.ErrNotRepo) {
		t.Errorf("err = %v, want ErrNotRepo", err)
	}
}

func TestRemoteURL_LocalRemoteHasNoHomePage(t *testing.T) {
	dir := newRepo(t)
	run(t, dir, "remote", "add", "origin", "/srv/git/repo.git")

	if _, err := git.RemoteURL(context.Background(), dir); err == nil {
		t.Error("RemoteURL for a local path remote succeeded, want an error")
	}
}
