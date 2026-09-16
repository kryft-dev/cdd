package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LastCommit returns the commit time of dir's HEAD, for dating a Visit
// seeded by a Scan. ok is false when dir is not a git repository (no .git
// entry, or git rejects it) or its branch is unborn (no commits yet); err is
// non-nil only when the check itself failed, such as a timeout.
func LastCommit(ctx context.Context, dir string) (time.Time, bool, error) {
	if !hasDotGit(dir) {
		return time.Time{}, false, nil
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "git", "-C", dir, "log", "-1", "--format=%cI")
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")

	out, err := cmd.Output()
	if err != nil {
		if cctx.Err() != nil {
			return time.Time{}, false, cctx.Err()
		}
		// Not a repository, or an unborn branch ("does not have any commits
		// yet"): both exit 128 with no stdout. Either way, there is no
		// commit to report.
		return time.Time{}, false, nil
	}

	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, false, nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false, err
	}

	return t, true, nil
}
