package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/project"
	"github.com/kryft-dev/cdd/internal/scan"
)

// newScanCmd builds "cdd scan [dir...]": seed History with one Visit per
// Project found at any depth below each dir (the home directory when none
// is given) and print a one-line summary.
func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan [dir...]",
		Short: "seed History with one Visit per git repository found below each dir (default: home)",
		RunE: func(cmd *cobra.Command, dirs []string) error {
			if len(dirs) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("cdd: locate home directory: %w", err)
				}
				dirs = []string{home}
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

			storePath, err := project.StorePath()
			if err != nil {
				return fmt.Errorf("cdd: locate the projects file: %w", err)
			}

			summary, err := scan.Run(cmd.Context(), dirs, cfg, hist, project.OpenStore(storePath))
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Seeded %d Visits across %d Projects\n", summary.Seeded, summary.Projects)
			return err
		},
	}
}
