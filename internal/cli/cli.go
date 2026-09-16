// Package cli wires cdd's command surface with github.com/spf13/cobra: the
// root command (bare behaves as pick), pick, init, scan, version, plus
// cobra's own help and completion commands.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/jump"
)

// Execute runs the CLI against os.Args and returns the process exit code.
func Execute() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// Run parses args against the cdd command tree, writing to stdout and
// stderr, and returns the process exit code: 0 on success, 130 silently on
// a cancelled Picker (jump.ErrCancelled), 1 with a stderr message on any
// other error, 2 with a stderr message on a usage error.
//
// Run is exported, rather than Execute alone, so tests can drive the CLI
// in-process with cobra's SetArgs and captured stdout/stderr instead of
// exec'ing a built binary.
func Run(args []string, stdout, stderr io.Writer) int {
	root := newRootCmd()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SilenceUsage = true
	root.SilenceErrors = true

	err := root.Execute()
	if err == nil {
		return 0
	}
	if errors.Is(err, jump.ErrCancelled) {
		return 130
	}

	fmt.Fprintln(stderr, err)
	if isUsageError(err) {
		return 2
	}
	return 1
}

// newRootCmd builds the cdd command tree: root (bare behaves as pick),
// pick, init, scan, version, plus cobra's help and completion commands.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cdd",
		Short: "cdd finds Projects under a Root and Jumps to the one you pick",
	}
	root.Flags().BoolP("version", "V", false, "print the cdd version and exit")

	root.RunE = func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Fprintln(cmd.OutOrStdout(), versionString())
			return nil
		}
		return pickRunE(cmd, nil)
	}

	root.AddCommand(newPickCmd(), newInitCmd(), newScanCmd(), newVersionCmd())

	// Initialized here, not left to Execute, so the command tree is
	// complete (including "help" and "completion") whenever init's
	// passthrough list is derived from it, even outside Execute.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	return root
}

// usageError marks err as a usage error: Run reports it with exit code 2
// instead of the default 1.
type usageError struct{ error }

// newUsageError wraps err so isUsageError recognizes it as a usage error.
func newUsageError(err error) error { return usageError{err} }

// isUsageError reports whether err should exit 2: either a command's own
// usageError, or one of cobra's own argument-parsing errors (an unknown
// command or an unknown flag).
func isUsageError(err error) bool {
	var ue usageError
	if errors.As(err, &ue) {
		return true
	}
	msg := err.Error()
	return strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "unknown shorthand flag") ||
		strings.HasPrefix(msg, "accepts ")
}

// isTerminal reports whether w is a terminal, so pickRunE knows whether to
// print the "cdd init <shell>" hint.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
