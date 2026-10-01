package picker

import (
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/action"
)

// vimKeys are the vim key map's own keys with what they do, for the help
// overlay.
var vimKeys = [][2]string{
	{"j / k", "move down / up"},
	{"g / G", "first / last row"},
	{"f or /", "focus the filter"},
	{"esc (filter)", "back to the list, keeping the query"},
	{"esc or q", "cancel"},
	{"?", "close this help"},
}

// helpAvailable reports whether `?` opens the help overlay: in the vim key
// map's list focus, unless an Action has taken the key.
func (m Model) helpAvailable() bool {
	_, taken := m.actions["?"]
	return m.vim && m.focus == focusList && !taken
}

// helpView renders the overlay of every key, navigation and Actions, in
// place of the whole frame. It is clipped to height, so a short terminal
// loses the end of the list and never the line that says how to close it.
func (m Model) helpView(t theme, width, height int) string {
	keyCol := 0
	for _, kv := range vimKeys {
		keyCol = max(keyCol, lipgloss.Width(kv[0]))
	}
	for _, a := range m.hints {
		keyCol = max(keyCol, lipgloss.Width(a.Key))
	}
	row := func(key, desc string) string {
		key = t.fg(t.muted).Bold(true).Render(key)
		desc = t.muted_().Render(truncateName(desc, max(width-keyCol-2, 1)))
		return padRightOn(plainStyle, key, keyCol) + "  " + desc
	}

	lines := []string{
		t.accentBold().Render("keys") + t.muted_().Render("  ? closes this help"),
		"",
		t.muted_().Render("navigation"),
	}
	for _, kv := range vimKeys {
		lines = append(lines, row(kv[0], kv[1]))
	}
	if len(m.hints) > 0 {
		lines = append(lines, "", t.muted_().Render("actions"))
	}
	for _, a := range m.hints {
		lines = append(lines, row(a.Key, a.Name+"  "+describe(a)))
	}
	return window(lines, 0, height)
}

// describe says in a few words what running a does.
func describe(a action.Action) string {
	switch {
	case a.Internal == action.InternalCopy:
		return "copies the path"
	case a.Run != "":
		return a.Run
	case a.Jump:
		return "jumps to the Project"
	}
	return ""
}
