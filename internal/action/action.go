// Package action defines the Actions the Picker can run on the selected
// Project: the built-in ones, a user's overrides of them from config.toml,
// and the Runner that starts their commands.
package action

import (
	"errors"
	"fmt"
	"runtime"
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
	// shell-quoted absolute path and "{remote}" for the shell-quoted home
	// page URL of its git remote, which fails the Action when it has none.
	// It may be empty only for an Action that
	// Jumps.
	Run string

	// Jump makes the Action Jump to the Project once Run exits.
	Jump bool

	// Detach starts Run in its own session without waiting, and leaves the
	// Picker open.
	Detach bool

	// Copy makes the Picker itself copy the Project's path to the clipboard
	// and stay open, with no Run. Only a built-in sets it, since the OSC 52
	// fallback has to go out through the Picker's own terminal. A user's
	// "run" or "jump" for the Action replaces it.
	Copy bool
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
	{Name: "files", Key: "ctrl+o", Run: Opener(runtime.GOOS) + " {path}", Detach: true},
	// The shell picks the editor each time the Action runs, so a change to
	// $VISUAL or $EDITOR needs no restart.
	{Name: "editor", Key: "ctrl+e", Run: "${VISUAL:-${EDITOR:-vi}} {path}"},
	{Name: "remote", Key: "ctrl+g", Run: Opener(runtime.GOOS) + " {remote}", Detach: true},
	{Name: "copy", Key: "ctrl+y", Copy: true},
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
// result in the order the Picker lists them: the user's own Actions first,
// in the order of order (the names of the user's tables as config.toml
// lists them; any it leaves out follow, sorted by name), then the
// built-ins in their own order. It fails on the first invalid Action with
// an *Error.
//
// A key a user table sets wins over an Action that only holds it by default,
// which is left unbound as if the user had set its key to "". Two Actions
// whose keys the user set to the same key are an error, as are two that
// both hold it by default.
//
// vim says whether the vim key map is on, the only one where a plain
// printable key may be bound.
func Merge(user map[string]Override, order []string, vim bool) ([]Action, error) {
	var out []Action
	index := make(map[string]int, len(user)+len(builtins))
	isBuiltin := make(map[string]bool, len(builtins))
	for _, a := range builtins {
		isBuiltin[a.Name] = true
	}

	add := func(name string) {
		if _, ok := user[name]; ok && !isBuiltin[name] {
			if _, seen := index[name]; !seen {
				index[name] = len(out)
				out = append(out, Action{Name: name})
			}
		}
	}
	for _, name := range order {
		add(name)
	}
	var rest []string
	for name := range user {
		rest = append(rest, name)
	}
	sort.Strings(rest)
	for _, name := range rest {
		add(name)
	}
	for _, a := range builtins {
		index[a.Name] = len(out)
		out = append(out, a)
	}

	for name, o := range user {
		a := &out[index[name]]
		if o.Key != nil {
			a.Key = *o.Key
		}
		if o.Run != nil {
			a.Run = *o.Run
			a.Copy = false
		}
		if o.Jump != nil {
			a.Jump = *o.Jump
			a.Copy = a.Copy && !a.Jump
		}
		if o.Detach != nil {
			a.Detach = *o.Detach
		}
	}

	owner := make(map[string]string, len(out))
	for i := range out {
		a := &out[i]
		if a.Run == "" && !a.Jump && !a.Copy {
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
		prev, clash := owner[key]
		switch {
		case !clash:
		case user[a.Name].Key != nil && user[prev].Key != nil:
			return nil, &Error{a.Name, "key", fmt.Errorf("%q is already bound to %q", key, prev)}
		case user[a.Name].Key != nil:
			out[index[prev]].Key = ""
		case user[prev].Key != nil:
			a.Key = ""
			continue
		default:
			return nil, &Error{a.Name, "key", fmt.Errorf("%q is already bound to %q", key, prev)}
		}
		owner[key] = a.Name
	}
	return out, nil
}
