// Package action defines the Actions the Picker can run on the selected
// Project: the built-in ones, a user's overrides of them from config.toml,
// and the Runner that starts their commands.
package action

import (
	"errors"
	"fmt"
	"slices"
	"sort"
)

// Action is a named command bound to a key in the Picker and run on the
// selected Project.
type Action struct {
	// Name identifies the Action in config.toml ([actions.<name>]).
	Name string

	// Key is the Picker key that runs the Action, in the canonical form of
	// a Bubble Tea key press ("ctrl+v"), or "" when the Action is unbound.
	Key string

	// Run is the shell command, with "{path}" standing for the Project's
	// shell-quoted absolute path. It may be empty only for an Action that
	// Jumps.
	Run string

	// Jump makes the Action Jump to the Project once Run exits.
	Jump bool

	// Detach starts Run in its own session without waiting, and leaves the
	// Picker open.
	Detach bool
}

// Override is the part of an Action a [actions.<name>] table sets. A nil
// field is one the table left out, so a built-in keeps its own value.
type Override struct {
	Key    *string `toml:"key"`
	Run    *string `toml:"run"`
	Jump   *bool   `toml:"jump"`
	Detach *bool   `toml:"detach"`
}

// builtins are the Actions cdd ships, in the order the Picker lists them.
// A built-in is registered by adding it here; a user's [actions.<name>]
// table with the same name overrides it field by field.
var builtins = []Action{
	{Name: "jump", Key: "enter", Jump: true},
}

// Builtins returns a copy of the built-in Actions.
func Builtins() []Action {
	return slices.Clone(builtins)
}

// Error is an invalid field of one Action. Field is the config key it
// concerns ("key", "run"), so a caller can point at its line.
type Error struct {
	Name  string
	Field string
	Err   error
}

func (e *Error) Error() string {
	return fmt.Sprintf("actions.%s: %s: %v", e.Name, e.Field, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// Merge applies user's overrides onto the built-in Actions and returns the
// result: the built-ins in their own order, then the user's other Actions
// sorted by name. It fails on the first invalid Action with an *Error.
//
// vim says whether the vim key map is on, the only one where a plain
// printable key may be bound.
func Merge(user map[string]Override, vim bool) ([]Action, error) {
	out := Builtins()
	index := make(map[string]int, len(out))
	for i, a := range out {
		index[a.Name] = i
	}

	var added []string
	for name := range user {
		if _, ok := index[name]; !ok {
			added = append(added, name)
		}
	}
	sort.Strings(added)
	for _, name := range added {
		index[name] = len(out)
		out = append(out, Action{Name: name})
	}

	for name, o := range user {
		a := &out[index[name]]
		if o.Key != nil {
			a.Key = *o.Key
		}
		if o.Run != nil {
			a.Run = *o.Run
		}
		if o.Jump != nil {
			a.Jump = *o.Jump
		}
		if o.Detach != nil {
			a.Detach = *o.Detach
		}
	}

	owner := make(map[string]string, len(out))
	for i := range out {
		a := &out[i]
		if a.Run == "" && !a.Jump {
			return nil, &Error{a.Name, "run", errors.New("a command is required unless jump is true")}
		}
		key, err := NormalizeKey(a.Key, vim)
		if err != nil {
			return nil, &Error{a.Name, "key", err}
		}
		a.Key = key
		if key == "" {
			continue
		}
		if prev, ok := owner[key]; ok {
			return nil, &Error{a.Name, "key", fmt.Errorf("%q is already bound to %q", key, prev)}
		}
		owner[key] = a.Name
	}
	return out, nil
}
