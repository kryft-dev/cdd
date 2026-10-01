package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kryft-dev/cdd/internal/action"
)

// resolveActions merges the [actions.*] tables onto the built-in Actions
// into c.ResolvedActions. data is config.toml's text, used only to name the
// line an invalid Action sits on.
func (c *Config) resolveActions(path string, data []byte) error {
	resolved, err := action.Merge(c.Actions, actionOrder(data), c.Keys.Vim)
	if err != nil {
		var ae *action.Error
		if errors.As(err, &ae) {
			if line := actionLine(data, ae.Name, ae.Field); line > 0 {
				return fmt.Errorf("config: %s: line %d: %w", path, line, err)
			}
		}
		return fmt.Errorf("config: %s: %w", path, err)
	}
	c.ResolvedActions = resolved
	return nil
}

var (
	tableHeader = regexp.MustCompile(`^\s*\[\s*([^\]]*?)\s*\]\s*(#.*)?$`)
	keyLine     = regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=`)
)

// actionOrder returns the names of the [actions.<name>] tables in the order
// data declares them, which the Picker lists them in. A name TOML lets be
// declared another way (dotted keys under a bare [actions] table) is left
// out, and Merge puts it after the rest.
func actionOrder(data []byte) []string {
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		m := tableHeader.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, ok := strings.CutPrefix(m[1], "actions.")
		if !ok {
			continue
		}
		if strings.HasPrefix(name, `"`) {
			unquoted, err := strconv.Unquote(name)
			if err != nil {
				continue
			}
			name = unquoted
		}
		names = append(names, name)
	}
	return names
}

// actionLine returns the 1-based line of field in the [actions.<name>]
// table of data, the table's header when it does not set field (an
// override can leave a built-in's key alone yet still clash), or 0 when
// there is no such table.
func actionLine(data []byte, name, field string) int {
	want := "actions." + name
	quoted := "actions." + strconv.Quote(name)

	header, inTable := 0, false
	for i, line := range strings.Split(string(data), "\n") {
		if m := tableHeader.FindStringSubmatch(line); m != nil {
			inTable = m[1] == want || m[1] == quoted
			if inTable && header == 0 {
				header = i + 1
			}
			continue
		}
		if m := keyLine.FindStringSubmatch(line); inTable && m != nil && m[1] == field {
			return i + 1
		}
	}
	return header
}
