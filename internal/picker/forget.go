package picker

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// askForget puts the forget prompt on the footer for row. Until the next
// key it is the only thing that reads one. With no Options.Forget there is
// nothing to honour a yes, so it asks nothing.
func (m Model) askForget(row Row) (tea.Model, tea.Cmd) {
	if m.forget != nil {
		m.confirm = &row
	}
	return m, nil
}

// updateKeyConfirm handles a key press while the forget prompt is up: y
// forgets, any other key cancels. The key is consumed either way, so a
// "y" or "n" is never typed into the filter.
func (m Model) updateKeyConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	row := *m.confirm
	m.confirm = nil
	if msg.String() != "y" {
		return m, nil
	}
	if err := m.forget(row.Project.Path); err != nil {
		m.message = fmt.Sprintf("forget: %v", err)
		return m, nil
	}
	// The row below, if any, takes the cursor's place, so the cursor stays
	// at its index until clampCursor pulls it back off the end.
	m.rows = slices.DeleteFunc(slices.Clone(m.rows), func(r Row) bool {
		return r.Project.Path == row.Project.Path
	})
	m.clampCursor()
	m.message, m.messageOK = "forgot "+row.Project.Name, true
	return m, nil
}
