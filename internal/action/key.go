package action

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// namedKeys are the multi-character key names the Picker recognises, as a
// Bubble Tea key press spells them.
var namedKeys = []string{
	"enter", "tab", "backspace", "delete", "insert", "esc", "space",
	"up", "down", "left", "right", "home", "end", "pgup", "pgdown",
}

// modifiers in the order a Bubble Tea key press spells them.
var modifiers = []string{"ctrl", "alt", "shift"}

// NormalizeKey checks that key names a key the Picker recognises and can
// bind, and returns it in the canonical spelling of a Bubble Tea key press
// (modifiers ordered ctrl, alt, shift). The empty key, an unbound Action,
// is valid.
//
// esc and ctrl+c can never be bound. A plain printable key would steal
// typing, so it is valid only when vim is true, where it applies in list
// focus.
func NormalizeKey(key string, vim bool) (string, error) {
	if key == "" {
		return "", nil
	}

	parts := strings.Split(key, "+")
	base := parts[len(parts)-1]
	if base == "" && len(parts) > 1 {
		base = "+" // "ctrl++" names the plus key itself
		parts = parts[:len(parts)-2]
	} else {
		parts = parts[:len(parts)-1]
	}

	var mods []string
	for _, m := range parts {
		if !slices.Contains(modifiers, m) || slices.Contains(mods, m) {
			return "", fmt.Errorf("unknown key %q", key)
		}
		mods = append(mods, m)
	}
	slices.SortFunc(mods, func(a, b string) int {
		return slices.Index(modifiers, a) - slices.Index(modifiers, b)
	})

	if err := checkBase(key, base, mods); err != nil {
		return "", err
	}

	canon := strings.Join(append(mods, base), "+")
	if canon == "esc" || canon == "ctrl+c" {
		return "", fmt.Errorf("%q can never be bound", canon)
	}
	if IsPrintable(canon) && !vim {
		return "", fmt.Errorf("%q would steal typing; plain keys need keys.vim = true", canon)
	}
	return canon, nil
}

// checkBase checks the last part of key, base, against its modifiers.
func checkBase(key, base string, mods []string) error {
	switch {
	case slices.Contains(namedKeys, base), isFunctionKey(base):
		return nil
	case utf8.RuneCountInString(base) != 1:
		return fmt.Errorf("unknown key %q", key)
	case slices.Contains(mods, "ctrl") && (base < "a" || base > "z"):
		return fmt.Errorf("ctrl takes a letter, got %q", key)
	case slices.Contains(mods, "shift"):
		return fmt.Errorf("shift takes a named key, got %q", key)
	}
	return nil
}

// isFunctionKey reports whether s is f1 to f20.
func isFunctionKey(s string) bool {
	n, err := strconv.Atoi(strings.TrimPrefix(s, "f"))
	return err == nil && strings.HasPrefix(s, "f") && n >= 1 && n <= 20
}

// IsPrintable reports whether key, in canonical form, is a plain typed
// character: one the filter would otherwise take as text.
func IsPrintable(key string) bool {
	return key == "space" || utf8.RuneCountInString(key) == 1
}
