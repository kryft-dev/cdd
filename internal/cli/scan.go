package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/scan"
)

// newScanCmd builds "cdd scan": seed History with one Visit per discovered
// Project and print a one-line summary.
func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "seed History with one Visit per discovered Project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			summary, err := scan.Run(cmd.Context(), cfg, hist)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Seeded %d Visits across %d Projects\n", summary.Seeded, summary.Projects)
			return err
		},
	}
}
