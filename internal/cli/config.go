package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kryft-dev/cdd"
	"github.com/kryft-dev/cdd/internal/config"
)

// newConfigCmd builds "cdd config": init, path and edit, which write, locate
// and open config.toml.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "create, locate, or edit config.toml",
	}
	cmd.AddCommand(newConfigInitCmd(), newConfigPathCmd(), newConfigEditCmd())
	return cmd
}

// newConfigInitCmd builds "cdd config init [--force]": write the commented
// example config to the config path, creating its directories. It refuses
// to replace a file that exists unless --force is given.
func newConfigInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "write the commented example config.toml (--force replaces an existing one)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			if err := writeExample(path, force); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace config.toml if it exists")
	return cmd
}

// newConfigPathCmd builds "cdd config path": print where config.toml lives,
// whether or not it exists.
func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "print the config.toml path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	}
}

// newConfigEditCmd builds "cdd config edit": open config.toml in $VISUAL,
// else $EDITOR, else vi, writing the example first when there is no file.
func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "open config.toml in $VISUAL or $EDITOR, creating it first if missing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			switch err := writeExample(path, false); {
			case err == nil:
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s\n", path); err != nil {
					return err
				}
			case !errors.Is(err, os.ErrExist):
				return err
			}

			// The shell resolves the editor, so a value such as "code -w"
			// works, as it does for the built-in editor Action.
			editor := exec.Command("sh", "-c", `${VISUAL:-${EDITOR:-vi}} "$1"`, "sh", path)
			editor.Stdin, editor.Stdout, editor.Stderr = os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr()
			if err := editor.Run(); err != nil {
				return fmt.Errorf("cdd config edit: %w", err)
			}
			return nil
		},
	}
}

// writeExample writes the example config to path, creating its directories.
// Unless force is set it fails with an error wrapping os.ErrExist when path
// exists, and never overwrites it.
func writeExample(path string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cdd config: %w", err)
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("cdd config: %s already exists (use --force to replace it): %w", path, os.ErrExist)
		}
		return fmt.Errorf("cdd config: %w", err)
	}
	if _, err := f.Write(cdd.ConfigExample); err != nil {
		_ = f.Close()
		return fmt.Errorf("cdd config: write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("cdd config: write %s: %w", path, err)
	}
	return nil
}
