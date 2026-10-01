// Package scan discovers Projects below the directories it is given and
// seeds History with one Visit per Project, so a fresh cdd installation has
// a starting order before any real Jump happens.
package scan

import (
	"context"
	"fmt"
	"slices"
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
	// Projects is the number of Projects discovered, Forgotten ones left out.
	Projects int
}

// Run discovers Projects at any depth below dirs, honoring cfg.Exclude and
// cfg.IncludeHidden, and seeds hist with one Visit per Project, except the
// ones store holds as Forgotten. Each Project's evidence time is
// git.LastCommit when the repository has a commit, else the Project
// directory's mtime.
//
// Whether a given Project actually gains a new Visit is entirely up to
// Seed's own idempotency rule (a Project with a newer real Visit is left
// alone); Run does not re-implement that rule, only reports how many
// Visits it produced.
func Run(ctx context.Context, dirs []string, cfg config.Config, hist *history.History, store *project.Store) (Summary, error) {
	projects, err := project.Discover(dirs, cfg.Exclude, cfg.IncludeHidden)
	if err != nil {
		return Summary{}, fmt.Errorf("scan: discover projects: %w", err)
	}

	marks, err := store.Marks()
	if err != nil {
		return Summary{}, fmt.Errorf("scan: %w", err)
	}
	projects = slices.DeleteFunc(projects, marks.IsForgotten)

	evidence, err := gatherEvidence(ctx, projects)
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{Projects: len(projects)}
	for _, p := range projects {
		seeded, err := seedOne(hist, p, evidence[p])
		if err != nil {
			return Summary{}, fmt.Errorf("scan: seed %q: %w", p, err)
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
func seedOne(hist *history.History, project string, at time.Time) (bool, error) {
	before, err := hist.Count(project)
	if err != nil {
		return false, err
	}
	if err := hist.Seed(project, at); err != nil {
		return false, err
	}
	after, err := hist.Count(project)
	if err != nil {
		return false, err
	}
	return after > before, nil
}
