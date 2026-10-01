package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

var (
	// ErrNotRepo is returned for a directory that is not a git repository.
	ErrNotRepo = errors.New("not a git repository")
	// ErrNoRemote is returned for a repository with no remote.
	ErrNoRemote = errors.New("no git remote")
)

// RemoteURL returns the home page URL (see WebURL) of dir's remote: origin,
// else the first remote git lists. It fails with ErrNotRepo or ErrNoRemote
// when there is nothing to open.
func RemoteURL(ctx context.Context, dir string) (string, error) {
	if !hasDotGit(dir) {
		return "", ErrNotRepo
	}

	out, err := gitOut(ctx, dir, "remote")
	if err != nil {
		return "", err
	}
	names := strings.Fields(out)
	if len(names) == 0 {
		return "", ErrNoRemote
	}
	name := names[0]
	for _, n := range names {
		if n == "origin" {
			name = n
		}
	}

	raw, err := gitOut(ctx, dir, "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return WebURL(raw)
}

// gitOut runs git in dir with args and returns its stdout.
func gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	return string(out), err
}
