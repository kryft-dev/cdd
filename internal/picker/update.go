package picker

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/action"
)

// Update handles one message: a key press, a status result landing, a
// terminal resize, the background colour report, or the deadline the first
// frame gives that report. It never blocks and never loses state (query,
// cursor, loaded statuses) across a resize.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusResultMsg:
		m.statuses[msg.path] = msg.status
		return m, nil

	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.paletteSettled = true
		return m, nil

	case paletteDeadlineMsg:
		m.paletteSettled = true
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

// updateKey dispatches a key press: to the Action bound to it, else by the
// active key map and focus. An Action's key overrides a navigation key of
// the same name.
func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.message, m.messageOK = "", false
	if m.confirm != nil {
		return m.updateKeyConfirm(msg)
	}
	if m.help {
		return m.updateKeyHelp(msg)
	}
	if a, ok := m.boundAction(msg.String()); ok {
		return m.runAction(a)
	}
	if m.vim {
		return m.updateKeyVim(msg)
	}
	return m.updateKeyDefault(msg)
}

// updateKeyDefault implements the default key map: typing filters, arrow
// keys (and ctrl+p/ctrl+n) move, esc cancels, ctrl+u clears.
func (m Model) updateKeyDefault(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "ctrl+p":
		m.moveCursor(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveCursor(1)
		return m, nil
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
// cancels.
func (m Model) updateKeyVim(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusFilter {
		switch msg.String() {
		case "esc":
			m.focus = focusList
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
	case "?":
		m.help = true
		return m, nil
	case "esc", "q", "ctrl+c":
		return m.cancel()
	}
	return m, nil
}

// updateKeyHelp handles a key press while the vim help overlay is open: ?,
// esc or q close it, ctrl+c still cancels, and nothing else does anything,
// so an Action key cannot fire unseen behind the overlay.
func (m Model) updateKeyHelp(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.help = false
	case "ctrl+c":
		return m.cancel()
	}
	return m, nil
}

// boundAction returns the Action bound to key. A plain printable key only
// counts in the vim key map's list focus, elsewhere it is typing.
func (m Model) boundAction(key string) (action.Action, bool) {
	a, ok := m.actions[key]
	if ok && action.IsPrintable(key) && (!m.vim || m.focus != focusList) {
		return action.Action{}, false
	}
	return a, ok
}

// runAction runs a on the row under the cursor, when there is one. A
// detached Action starts and the Picker stays open, showing the failure on
// the footer line if it did not start; any other quits with the Action as
// the Choice, for the caller to run once the screen is restored.
func (m Model) runAction(a action.Action) (tea.Model, tea.Cmd) {
	rows := m.visibleRows()
	if m.cursor < 0 || m.cursor >= len(rows) {
		return m, nil
	}
	row := rows[m.cursor].row

	switch a.Internal {
	case action.InternalCopy:
		return m.copyPath(a, row.Project.Path)
	case action.InternalForget:
		return m.askForget(row)
	}
	if !a.Detach {
		m.chosen = true
		m.chosenRow = row
		m.chosenAction = &a
		m.quitting = true
		return m, tea.Quit
	}
	if err := m.runner.Start(a, row.Project.Path); err != nil {
		m.message = fmt.Sprintf("%s: %v", a.Name, err)
	}
	return m, nil
}

// copyPath copies path with the clipboard program there is, or else asks
// Bubble Tea for the OSC 52 escape, which reaches the terminal even over
// SSH. Either way the Picker stays open and the footer says "copied".
func (m Model) copyPath(a action.Action, path string) (tea.Model, tea.Cmd) {
	copied, err := m.copy(path)
	if err != nil {
		m.message = fmt.Sprintf("%s: %v", a.Name, err)
		return m, nil
	}
	m.message, m.messageOK = "copied "+path, true
	if copied {
		return m, nil
	}
	return m, tea.SetClipboard(path)
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
	n := len(m.visibleRows())
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

// cancel quits without a choice.
func (m Model) cancel() (tea.Model, tea.Cmd) {
	m.chosen = false
	m.quitting = true
	return m, tea.Quit
}
