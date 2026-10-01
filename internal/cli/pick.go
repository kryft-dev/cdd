package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/config"
	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/jump"
	"github.com/kryft-dev/cdd/internal/picker"
	"github.com/kryft-dev/cdd/internal/project"
)

// initHint is printed to stderr, after the chosen path, when stdout is a
// terminal and no Wrapper is forwarding the path into a Jump.
const initHint = `cdd: no Wrapper installed, so this path was only printed.
cdd: add "cdd init <shell> | source" (or the "eval" form) to your shell config so cdd can Jump.`

// newPickCmd builds "cdd pick".
func newPickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pick",
		Short: "prints the chosen Project's path; the Wrapper turns it into a Jump",
		Args:  cobra.NoArgs,
		RunE:  pickRunE,
	}
}

// pickRunE implements both "cdd pick" and the bare "cdd" root command:
// load config, open History and the projects file at their default paths, resolve a Project via
// jump.Resolve and picker.Run, and print the chosen absolute path plus a newline to stdout and nothing else.
// An Action that does not Jump prints nothing, so the Wrapper has nothing
// to cd to.
//
// A cancelled Picker returns jump.ErrCancelled, which Run reports as exit
// 130 with no message, and an Action's non-zero exit status is passed on
// through jump.ExitError. Any other error is reported by Run as exit 1 with
// the error's message on stderr.
func pickRunE(cmd *cobra.Command, _ []string) error {
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

	abs, err := jump.Resolve(cmd.Context(), cfg, hist, project.OpenStore(storePath), picker.Run, action.ExecRunner{})
	if err != nil {
		return err
	}

	if abs == "" {
		return nil // an Action ran and does not Jump
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
