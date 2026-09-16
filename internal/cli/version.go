package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version, commit, and date are set at build time via -ldflags (see
// .goreleaser.yaml). They default to values that make an unflagged build
// identify itself as a dev build.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// newVersionCmd builds "cdd version", which prints the same output as
// "cdd --version".
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the cdd version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), versionString())
			return nil
		},
	}
}

// versionString formats the version, commit, and date set by ldflags. When
// none of them were set, it reports "cdd dev (dev)".
func versionString() string {
	if version == "dev" && commit == "" && date == "" {
		return "cdd dev (dev)"
	}
	return fmt.Sprintf("cdd %s (%s %s)", version, commit, date)
}
