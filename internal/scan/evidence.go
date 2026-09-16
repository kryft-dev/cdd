package scan

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/project"
)

// workers bounds the number of concurrent git calls a Scan makes while
// gathering each Project's evidence time, so scanning a large Root never
// spawns one git process per Project at once.
const workers = 8

// gatherEvidence returns each Project's evidence time, keyed by Rel: the
// Project's git.LastCommit time when it is a repository with a commit,
// else its directory's mtime. Git calls are bounded to workers at a time.
func gatherEvidence(ctx context.Context, root string, projects []project.Project) (map[string]time.Time, error) {
	type result struct {
		rel string
		at  time.Time
		err error
	}

	sem := make(chan struct{}, workers)
	results := make(chan result, len(projects))
	var wg sync.WaitGroup

	for _, p := range projects {
		wg.Add(1)
		go func(p project.Project) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			at, err := evidenceTime(ctx, p.Abs(root))
			results <- result{rel: p.Rel(), at: at, err: err}
		}(p)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	evidence := make(map[string]time.Time, len(projects))
	for r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("scan: evidence time for %q: %w", r.rel, r.err)
		}
		evidence[r.rel] = r.at
	}
	return evidence, nil
}

// evidenceTime returns dir's evidence time: its git.LastCommit time when
// dir is a repository with a commit, else dir's mtime. The result is
// truncated to UTC second precision, matching History's own line format, so
// repeated Scans compare equal rather than drifting on sub-second noise.
func evidenceTime(ctx context.Context, dir string) (time.Time, error) {
	commit, ok, err := git.LastCommit(ctx, dir)
	if err != nil {
		return time.Time{}, err
	}
	if ok {
		return commit.UTC().Truncate(time.Second), nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime().UTC().Truncate(time.Second), nil
}
