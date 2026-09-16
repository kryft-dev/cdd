// Package scan discovers Projects under a Root and seeds History with one
// Visit per Project, so a fresh cdd installation has a starting order
// before any real Jump happens.
package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/project"
)

// Summary reports what a Scan did, for the cli to print as "Seeded N
// Visits across M Projects".
type Summary struct {
	// Seeded is the number of Visits actually appended to History.
	Seeded int
	// Projects is the number of Projects discovered under Root.
	Projects int
}

// Run discovers Projects under cfg.Root, honoring cfg.Exclude and
// cfg.IncludeHidden, and seeds hist with one Visit per Project. Each
// Project's evidence time is git.LastCommit when the Project is a
// repository with a commit, else the Project directory's mtime.
//
// Whether a given Project actually gains a new Visit is entirely up to
// Seed's own idempotency rule (a Project with a newer real Visit is left
// alone); Run does not re-implement that rule, only reports how many
// Visits it produced.
func Run(ctx context.Context, cfg config.Config, hist *history.History) (Summary, error) {
	projects, err := project.Discover(cfg.Root, cfg.Exclude, cfg.IncludeHidden)
	if err != nil {
		return Summary{}, fmt.Errorf("scan: discover projects: %w", err)
	}

	evidence, err := gatherEvidence(ctx, cfg.Root, projects)
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{Projects: len(projects)}
	for _, p := range projects {
		seeded, err := seedOne(hist, p.Rel(), evidence[p.Rel()])
		if err != nil {
			return Summary{}, fmt.Errorf("scan: seed %q: %w", p.Rel(), err)
		}
		if seeded {
			summary.Seeded++
		}
	}

	return summary, nil
}

// seedOne calls Seed for project and reports whether it actually appended a
// Visit, by comparing History's Visit count for project before and after.
// This leans on history.Count rather than re-implementing Seed's
// idempotency rule.
func seedOne(hist *history.History, rel string, at time.Time) (bool, error) {
	before, err := hist.Count(rel)
	if err != nil {
		return false, err
	}
	if err := hist.Seed(rel, at); err != nil {
		return false, err
	}
	after, err := hist.Count(rel)
	if err != nil {
		return false, err
	}
	return after > before, nil
}
