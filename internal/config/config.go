// Package config loads cdd's configuration: the Exclude globs and
// hidden-directory handling a Scan walks with, and the History and Picker
// settings. Every key is optional, and so is the file itself.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/kryft-dev/cdd/internal/action"
)

// Config is cdd's configuration, decoded from config.toml.
type Config struct {
	// Exclude holds filepath.Match patterns for directories a Scan skips,
	// with everything below them: a pattern containing "/" is matched
	// against a directory's absolute path, any other against its name
	// alone. A leading "~" is expanded to the user's home directory, but
	// "$VAR" is left as-is.
	Exclude []string `toml:"exclude"`

	// IncludeHidden, when false (the default), makes a Scan skip hidden
	// directories.
	IncludeHidden bool `toml:"include_hidden"`

	// History configures the ordered record of Visits.
	History History `toml:"history"`

	// Keys configures the Picker's key map.
	Keys Keys `toml:"keys"`

	// Picker configures the Picker's appearance.
	Picker Picker `toml:"picker"`

	// Actions holds the [actions.<name>] tables: new Actions, and
	// overrides of built-in ones by name.
	Actions map[string]action.Override `toml:"actions"`

	// ResolvedActions is every Action the Picker binds: the built-ins with
	// Actions applied on top. LoadFrom fills it in.
	ResolvedActions []action.Action `toml:"-"`
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

// Picker configures the Picker's appearance.
type Picker struct {
	// Layout selects which layout the Picker draws. "list", the default
	// and only layout today, is a flat fzf-style list with the filter
	// prompt below it and a background-highlighted selected row.
	Layout string `toml:"layout"`

	// Hints, when true (the default), draws the key-hint line under the
	// list, built from the bound Actions. When false the line shows only
	// the match count and any message.
	Hints bool `toml:"hints"`
}

// pickerLayouts are the values Picker.Layout accepts, in the order the
// error message lists them.
var pickerLayouts = []string{"list"}

// defaultConfig returns a Config with every default applied, before a
// config.toml's fields are decoded on top of it.
func defaultConfig() Config {
	return Config{
		Exclude:       []string{},
		IncludeHidden: false,
		History:       History{MaxVisits: 1000},
		Keys:          Keys{Vim: false},
		Picker:        Picker{Layout: "list", Hints: true},
	}
}

// Path returns where cdd's configuration lives:
// $XDG_CONFIG_HOME/cdd/config.toml, defaulting to ~/.config/cdd/config.toml
// when XDG_CONFIG_HOME is unset.
func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "cdd", "config.toml"), nil
}

// Load reads cdd's configuration from Path.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return LoadFrom(path)
}

// LoadFrom reads cdd's configuration from path. A missing file is every
// default. It is exported separately from Load so tests can point it at a
// fixture file.
func LoadFrom(path string) (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := cfg.resolveActions(path, nil)
			return cfg, err
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	dec := toml.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %s: %w", path, describeDecodeError(err))
	}

	if err := cfg.validate(path); err != nil {
		return Config{}, err
	}
	if err := cfg.resolveActions(path, data); err != nil {
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
// enforce, expanding "~" in each Exclude pattern on the way: every Exclude
// pattern must be well formed; History.MaxVisits must be at least 1;
// Picker.Layout must name a known layout.
func (c *Config) validate(path string) error {
	for i, p := range c.Exclude {
		expanded, err := expandHome(p)
		if err != nil {
			return fmt.Errorf("config: %s: expand exclude pattern %q: %w", path, p, err)
		}
		if _, err := filepath.Match(expanded, ""); err != nil {
			return fmt.Errorf("config: %s: invalid exclude pattern %q: %w", path, p, err)
		}
		c.Exclude[i] = expanded
	}

	if c.History.MaxVisits < 1 {
		return fmt.Errorf("config: %s: history.max_visits must be >= 1, got %d", path, c.History.MaxVisits)
	}

	if !slices.Contains(pickerLayouts, c.Picker.Layout) {
		quoted := make([]string, len(pickerLayouts))
		for i, l := range pickerLayouts {
			quoted[i] = strconv.Quote(l)
		}
		return fmt.Errorf("config: %s: picker.layout must be %s, got %q", path, strings.Join(quoted, " or "), c.Picker.Layout)
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
