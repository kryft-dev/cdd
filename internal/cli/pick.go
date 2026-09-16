package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
)

// initHint is printed to stderr, after the chosen path, when stdout is a
// terminal and no Wrapper is forwarding the path into a Jump.
const initHint = `cdd: no Wrapper installed, so this path was only printed.
cdd: add "cdd init <shell> | source" (or the "eval" form) to your shell config so cdd can Jump.`

// newPickCmd builds "cdd pick [query]".
func newPickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pick [query]",
		Short: "prints the chosen Project's path; the Wrapper turns it into a Jump",
		Args:  cobra.MaximumNArgs(1),
		RunE:  pickRunE,
	}
}

// pickRunE implements both "cdd pick [query]" and the bare "cdd" root
// command: load config, open History at its default path, resolve the
// query to a Project via jump.Resolve and picker.Run, and print the
// chosen absolute path plus a newline to stdout and nothing else.
//
// A cancelled Picker returns jump.ErrCancelled, which Run reports as exit
// 130 with no message. Any other error is reported by Run as exit 1 with
// the error's message on stderr.
func pickRunE(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
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

	abs, err := jump.Resolve(cmd.Context(), cfg, hist, query, picker.Run)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, abs); err != nil {
		return err
	}
	if isTerminal(out) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), initHint)
	}
	return nil
}
