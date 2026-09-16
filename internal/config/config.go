// Package config loads cdd's configuration: the Root to search for Kinds
// and Projects, Exclude globs, hidden-directory handling, and the History
// and Picker key settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Config is cdd's configuration, decoded from config.toml.
type Config struct {
	// Root is the top-level directory whose Kinds are searched for
	// Projects. Required; a leading "~" is expanded to the user's home
	// directory, but "$VAR" is left as-is.
	Root string `toml:"root"`

	// Exclude holds paths relative to Root, matched with path.Match
	// semantics, that are skipped when discovering Kinds and Projects.
	Exclude []string `toml:"exclude"`

	// IncludeHidden, when false (the default), excludes hidden
	// directories at both Kind and Project level.
	IncludeHidden bool `toml:"include_hidden"`

	// History configures the ordered record of Visits.
	History History `toml:"history"`

	// Keys configures the Picker's key map.
	Keys Keys `toml:"keys"`
}

// History configures cdd's History of Visits.
type History struct {
	// MaxVisits is the maximum number of Visits kept in History. Must be
	// at least 1.
	MaxVisits int `toml:"max_visits"`
}

// Keys configures the Picker's key map.
type Keys struct {
	// Vim, when true, makes the Picker open with the list focused and use
	// vim-style keys: j/k move, f focuses the filter, esc returns to the
	// list.
	Vim bool `toml:"vim"`
}

// defaultConfig returns a Config with every default applied, before a
// config.toml's fields are decoded on top of it.
func defaultConfig() Config {
	return Config{
		Exclude:       []string{},
		IncludeHidden: false,
		History:       History{MaxVisits: 1000},
		Keys:          Keys{Vim: false},
	}
}

// exampleConfigBody is the example config.toml shown in errors and printed
// by ExampleConfig.
const exampleConfigBody = `root = "~/Developer"      # required, no default
exclude = []              # paths relative to Root, glob-matched
include_hidden = false
[history]
max_visits = 1000         # must be >= 1
[keys]
vim = false
`

// ExampleConfig returns an example config.toml, for the cli package to print
// when Load or LoadFrom fails.
func ExampleConfig() string {
	return exampleConfigBody
}

// Load reads cdd's configuration from $XDG_CONFIG_HOME/cdd/config.toml,
// defaulting to ~/.config/cdd/config.toml when XDG_CONFIG_HOME is unset.
func Load() (Config, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("config: determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return LoadFrom(filepath.Join(dir, "cdd", "config.toml"))
}

// LoadFrom reads cdd's configuration from path. It is exported separately
// from Load so tests can point it at a fixture file.
func LoadFrom(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("config: %s not found\n\nExample config.toml:\n\n%s", path, exampleConfigBody)
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := defaultConfig()
	dec := toml.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %s: %w", path, describeDecodeError(err))
	}

	if err := cfg.validate(path); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// describeDecodeError rewrites a toml decode error so it names the
// offending key and line. For an unknown-key error (StrictMissingError) it
// reports each missing field's key and position; other decode errors
// (wrong types) already carry a key and position via DecodeError.String.
func describeDecodeError(err error) error {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) {
		msgs := make([]string, len(strict.Errors))
		for i, e := range strict.Errors {
			row, col := e.Position()
			msgs[i] = fmt.Sprintf("unknown key %q at line %d column %d", e.Key(), row, col)
		}
		return errors.New(strings.Join(msgs, "; "))
	}

	var decodeErr *toml.DecodeError
	if errors.As(err, &decodeErr) {
		row, col := decodeErr.Position()
		return fmt.Errorf("%w (line %d column %d)", err, row, col)
	}

	return err
}

// validate checks the decoded Config against the rules Load and LoadFrom
// enforce: Root is required, expanded, and must be an existing directory;
// History.MaxVisits must be at least 1.
func (c *Config) validate(path string) error {
	if c.Root == "" {
		return fmt.Errorf("config: %s: root is required\n\nExample config.toml:\n\n%s", path, exampleConfigBody)
	}

	root, err := expandHome(c.Root)
	if err != nil {
		return fmt.Errorf("config: %s: expand root %q: %w", path, c.Root, err)
	}
	c.Root = root

	info, err := os.Stat(c.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config: %s: root %q does not exist", path, c.Root)
		}
		return fmt.Errorf("config: %s: root %q: %w", path, c.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("config: %s: root %q is not a directory", path, c.Root)
	}

	if c.History.MaxVisits < 1 {
		return fmt.Errorf("config: %s: history.max_visits must be >= 1, got %d", path, c.History.MaxVisits)
	}

	return nil
}

// expandHome expands a leading "~" in path to the user's home directory.
// "$VAR" references are left as-is.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
