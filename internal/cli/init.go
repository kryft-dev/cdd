package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd/internal/shell"
)

// newInitCmd builds "cdd init <shell>": exactly one argument in fish,
// bash, or zsh, else a usage error (exit 2) listing them.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <shell>",
		Short: "print the Wrapper for fish, bash, or zsh",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return newUsageError(fmt.Errorf("cdd init: expected exactly one shell argument: fish, bash, zsh"))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := shell.Script(args[0], passthrough(cmd.Root()))
			if err != nil {
				return newUsageError(err)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), script)
			return err
		},
	}
}

// passthrough lists every command name in root's tree except "pick", the
// Wrapper's pass-through list. It is derived from the cobra command tree
// rather than hand maintained, so "help" and "completion" pass through
// without extra work.
func passthrough(root *cobra.Command) []string {
	var names []string
	for _, c := range root.Commands() {
		if c.Name() == "pick" {
			continue
		}
		names = append(names, c.Name())
	}
	return names
}
