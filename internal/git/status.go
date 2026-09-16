// Package git is where all shelling out to git lives. Nothing else in cdd
// runs the git binary directly.
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// timeout bounds every git invocation this package makes, so a slow or
// hanging git call never stalls a Scan or the Picker.
const timeout = 2 * time.Second

// Kind classifies the outcome of a Status check.
type Kind int

const (
	// Found means dir is a git repository; the rest of Status describes it.
	Found Kind = iota
	// NotRepo means dir does not contain a .git file or directory.
	NotRepo
	// Unknown means the check could not complete: a timeout, or a git error
	// other than "not a repository".
	Unknown
)

// String renders k for logs and debugging.
func (k Kind) String() string {
	switch k {
	case Found:
		return "found"
	case NotRepo:
		return "not-repo"
	case Unknown:
		return "unknown"
	default:
		return "kind(?)"
	}
}

// State is a Project's worktree state: whether it has changes relative to
// its last commit. It is only meaningful when Status.Kind is Found.
type State int

const (
	Clean State = iota
	Dirty
)

// String renders s for logs and debugging.
func (s State) String() string {
	if s == Dirty {
		return "dirty"
	}
	return "clean"
}

// Status is the result of checking a Project's git status. When Kind is not
// Found, every other field is the zero value.
type Status struct {
	Kind Kind

	State       State
	Untracked   bool
	Ahead       int
	Behind      int
	HasUpstream bool
	Branch      string
}

// GetStatus reports dir's git status: worktree state (clean, dirty,
// untracked) and, when dir has an upstream, how far it has diverged from it.
//
// Note: the ticket names this function "Status", but Go does not allow a
// function and a type to share an identifier in the same package, so the
// function is named GetStatus and the result type keeps the name Status.
//
// Only a directory that itself contains a .git file or directory (a
// repository root, or a linked worktree) is treated as a repository; a
// subdirectory of one reports NotRepo rather than the enclosing repo's
// status, so git is never allowed to discover a repo by walking up.
func GetStatus(ctx context.Context, dir string) (Status, error) {
	if !hasDotGit(dir) {
		return Status{Kind: NotRepo}, nil
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "git", "-C", dir,
		"status", "--porcelain=v2", "--branch", "-unormal", "-z")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")

	out, err := cmd.Output()
	if err != nil {
		if cctx.Err() != nil {
			return Status{Kind: Unknown}, nil
		}
		if isNotRepo(err) {
			return Status{Kind: NotRepo}, nil
		}
		return Status{Kind: Unknown}, nil
	}

	return parseStatus(out), nil
}

// hasDotGit reports whether dir directly contains a .git entry, file or
// directory.
func hasDotGit(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

// isNotRepo reports whether err is git exiting 128 with "fatal:" on stderr,
// the signature of "not a repository" (and related: bare repo, missing dir).
func isNotRepo(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	return exitErr.ExitCode() == 128 && bytes.Contains(exitErr.Stderr, []byte("fatal:"))
}
