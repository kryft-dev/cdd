package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// version, commit, and date are set at build time via -ldflags (see
// .goreleaser.yaml). They default to values that make an unflagged build
// fall back to the build information Go embeds in every binary.
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
			_, err := fmt.Fprintln(cmd.OutOrStdout(), versionString())
			return err
		},
	}
}

// versionString formats the version from ldflags when GoReleaser set them,
// otherwise from the build information Go embeds: the module version for
// a `go install module@tag` build, the VCS revision for a build from a
// checkout, and "cdd dev (dev)" when neither is known.
func versionString() string {
	info, _ := debug.ReadBuildInfo()
	return formatVersion(version, commit, date, info)
}

// formatVersion is the pure core of versionString, split out so tests can
// feed it ldflags values and build information directly.
func formatVersion(version, commit, date string, info *debug.BuildInfo) string {
	if version != "dev" || commit != "" || date != "" {
		return fmt.Sprintf("cdd %s (%s %s)", version, commit, date)
	}
	if info == nil {
		return "cdd dev (dev)"
	}

	if v := info.Main.Version; v != "" && v != "(devel)" {
		return fmt.Sprintf("cdd %s (go install)", strings.TrimPrefix(v, "v"))
	}

	rev, when, modified := vcsSettings(info)
	if rev == "" {
		return "cdd dev (dev)"
	}
	if modified {
		rev += "+dirty"
	}
	if when == "" {
		return fmt.Sprintf("cdd dev (%s)", rev)
	}
	return fmt.Sprintf("cdd dev (%s %s)", rev, when)
}

// vcsSettings extracts the short revision, commit date, and dirty flag that
// Go records when building from a version-controlled checkout.
func vcsSettings(info *debug.BuildInfo) (rev, when string, modified bool) {
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
		case "vcs.time":
			when = s.Value
			if len(when) > 10 {
				when = when[:10]
			}
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return rev, when, modified
}
