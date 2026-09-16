package picker

import (
	tea "charm.land/bubbletea/v2"
)

// Update handles one message: a key press, a status result landing, a
// terminal resize, or the background colour report. It never blocks and
// never loses state (query, cursor, loaded statuses) across a resize.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusResultMsg:
		m.statuses[msg.path] = msg.status
		return m, nil

	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.themeSet = true
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampCursor()
		return m, nil

	case tea.KeyPressMsg:
		return m.updateKey(msg)
	}

	return m, nil
}

// updateKey dispatches a key press by the active key map and focus.
func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.vim {
		return m.updateKeyVim(msg)
	}
	return m.updateKeyDefault(msg)
}

// updateKeyDefault implements the default key map: typing filters, arrow
// keys (and ctrl+p/ctrl+n) move, enter chooses, esc cancels, ctrl+u clears.
func (m Model) updateKeyDefault(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "ctrl+p":
		m.moveCursor(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveCursor(1)
		return m, nil
	case "enter":
		return m.choose()
	case "esc", "ctrl+c":
		return m.cancel()
	case "ctrl+u":
		m.query = ""
		m.cursor = 0
		return m, nil
	case "backspace":
		m.backspace()
		return m, nil
	default:
		if msg.Text != "" {
			m.query += msg.Text
			m.cursor = 0
		}
		return m, nil
	}
}

// updateKeyVim implements the vim key map. The list is focused on open;
// j/k move, g/G jump to the ends, f or / focuses the filter, esc in the
// filter returns to the list keeping the query, esc or q on the list
// cancels, enter chooses from either mode.
func (m Model) updateKeyVim(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusFilter {
		switch msg.String() {
		case "esc":
			m.focus = focusList
			return m, nil
		case "enter":
			return m.choose()
		case "backspace":
			m.backspace()
			return m, nil
		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.cursor = 0
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "j":
		m.moveCursor(1)
		return m, nil
	case "k":
		m.moveCursor(-1)
		return m, nil
	case "g":
		m.cursor = 0
		return m, nil
	case "G":
		m.cursor = m.lastIndex()
		return m, nil
	case "f", "/":
		m.focus = focusFilter
		return m, nil
	case "enter":
		return m.choose()
	case "esc", "q", "ctrl+c":
		return m.cancel()
	}
	return m, nil
}

// moveCursor shifts the cursor by delta rows, clamped to the visible range.
func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	m.clampCursor()
}

// clampCursor keeps the cursor within the currently visible rows, which can
// shrink after a filter change or a resize.
func (m *Model) clampCursor() {
	last := m.lastIndex()
	if m.cursor > last {
		m.cursor = last
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// lastIndex is the index of the last visible row, or 0 when there are none.
func (m Model) lastIndex() int {
	n := len(flatten(m.visibleGroups()))
	if n == 0 {
		return 0
	}
	return n - 1
}

// backspace removes the last rune from the filter query.
func (m *Model) backspace() {
	if m.query == "" {
		return
	}
	r := []rune(m.query)
	m.query = string(r[:len(r)-1])
	m.cursor = 0
}

// choose selects the row under the cursor, when there is one, and quits.
func (m Model) choose() (tea.Model, tea.Cmd) {
	rows := flatten(m.visibleGroups())
	if m.cursor >= 0 && m.cursor < len(rows) {
		m.chosen = true
		m.chosenRow = rows[m.cursor].row
	}
	m.quitting = true
	return m, tea.Quit
}

// cancel quits without a choice.
func (m Model) cancel() (tea.Model, tea.Cmd) {
	m.chosen = false
	m.quitting = true
	return m, tea.Quit
}
