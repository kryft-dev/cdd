package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/project"
)

// newAddCmd builds "cdd add [dir]": make dir (the current directory by
// default) a Project even without a .git, bring it back if it was Forgotten,
// and Record a Visit so it shows in the Picker at once.
func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [dir]",
		Short: "make a directory a Project, git repository or not (default: current directory)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := absDir(args)
			if err != nil {
				return err
			}
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				return fmt.Errorf("cdd add: %q is not a directory", dir)
			}

			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.Add(dir); err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			histPath, err := history.DefaultPath()
			if err != nil {
				return fmt.Errorf("cdd: locate History: %w", err)
			}
			hist, err := history.Open(histPath, cfg.History.MaxVisits)
			if err != nil {
				return err
			}
			if err := hist.Record(dir); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %s\n", dir)
			return err
		},
	}
}

// newForgetCmd builds "cdd forget [dir]": hide a Project, git or not, from
// the Picker and from Scan until it is added again. The directory need not
// exist any more.
func newForgetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "forget [dir]",
		Short: "hide a Project from the Picker and Scan (default: current directory)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := absDir(args)
			if err != nil {
				return err
			}
			store, err := openStore()
			if err != nil {
				return err
			}
			if err := store.Forget(dir); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Forgot %s\n", dir)
			return err
		},
	}
}

// absDir returns the cleaned absolute path of the one optional argument, or
// of the current directory when there is none.
func absDir(args []string) (string, error) {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("cdd: resolve %q: %w", dir, err)
	}
	return abs, nil
}

// openStore opens the projects file at its default path.
func openStore() (*project.Store, error) {
	path, err := project.StorePath()
	if err != nil {
		return nil, fmt.Errorf("cdd: locate the projects file: %w", err)
	}
	return project.OpenStore(path), nil
}
