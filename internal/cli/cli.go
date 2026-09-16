// Package cli is the stub command-line entry point for cdd. It handles the
// "version" subcommand only; full CLI wiring (cobra, Root, Kind, Project,
// Jump, Visit, History, Scan, Picker, Wrapper commands) lands in a later
// ticket.
package cli

import (
	"fmt"
	"os"
)

// version, commit, and date are set at build time via -ldflags. They default
// to values that make an unflagged build identify itself as a dev build.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

// Execute runs the CLI and returns the process exit code. It handles
// "version" and prints a usage line to stderr for anything else.
func Execute() int {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "version" {
		fmt.Println(versionString())
		return 0
	}

	fmt.Fprintln(os.Stderr, "usage: cdd version")
	return 2
}

// versionString formats the version, commit, and date set by ldflags. When
// none of them were set, it reports "cdd dev (dev)".
func versionString() string {
	if version == "dev" && commit == "" && date == "" {
		return "cdd dev (dev)"
	}
	return fmt.Sprintf("cdd %s (%s %s)", version, commit, date)
}
