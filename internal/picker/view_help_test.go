package picker_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/picker"
)

var (
	question = tea.KeyPressMsg{Code: '?', Text: "?"}
	escape   = tea.KeyPressMsg{Code: tea.KeyEscape}
)

func TestHelp_VimQuestionMarkOpensAndClosesTheOverlay(t *testing.T) {
	m := actionModel(&fakeRunner{}, true, action.Builtins()...)

	m, _ = press(m, question)
	open := plain(m.View().Content)
	for _, want := range []string{"j / k", "move down / up", "esc or q", "cancel", "actions", "ctrl+o", "files", "ctrl+e"} {
		if !strings.Contains(open, want) {
			t.Errorf("overlay lacks %q:\n%s", want, open)
		}
	}
	if strings.Contains(open, "alpha") {
		t.Errorf("overlay still shows the list:\n%s", open)
	}

	m, _ = press(m, question)
	if closed := plain(m.View().Content); !strings.Contains(closed, "alpha") || strings.Contains(closed, "navigation") {
		t.Errorf("? did not close the overlay:\n%s", closed)
	}
}

func TestHelp_EscAndQCloseItInsteadOfCancelling(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{escape, {Code: 'q', Text: "q"}} {
		m := actionModel(&fakeRunner{}, true, action.Builtins()...)
		m, _ = press(m, question)

		m, cmd := press(m, key)

		if cmd != nil || strings.Contains(plain(m.View().Content), "navigation") {
			t.Errorf("%v: Cmd = %v, want the overlay closed and the Picker open", key, cmd != nil)
		}
	}
}

func TestHelp_SwallowsActionKeysButCtrlCStillCancels(t *testing.T) {
	m := actionModel(&fakeRunner{}, true, action.Builtins()...)
	m, _ = press(m, question)

	m, cmd := press(m, enter)
	if _, chosen := m.Chosen(); chosen || cmd != nil {
		t.Fatalf("enter under the overlay chose a Project")
	}
	if !strings.Contains(plain(m.View().Content), "navigation") {
		t.Errorf("a stray key closed the overlay")
	}

	_, cmd = press(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Errorf("ctrl+c under the overlay did not quit")
	}
}

func TestHelp_OnlyInVimListFocus(t *testing.T) {
	// The default key map: ? is typing.
	m := actionModel(&fakeRunner{}, false, action.Builtins()...)
	m, _ = press(m, question)
	if v := plain(m.View().Content); strings.Contains(v, "navigation") || !strings.Contains(v, "❯ ?") {
		t.Errorf("default map: ? should type into the filter:\n%s", v)
	}

	// The vim filter focus: ? is typing too.
	m = actionModel(&fakeRunner{}, true, action.Builtins()...)
	m, _ = press(m, tea.KeyPressMsg{Code: 'f', Text: "f"})
	m, _ = press(m, question)
	if v := plain(m.View().Content); strings.Contains(v, "navigation") || !strings.Contains(v, "❯ ?") {
		t.Errorf("vim filter focus: ? should type into the filter:\n%s", v)
	}
}

func TestHelp_AnActionOnQuestionMarkWinsTheKey(t *testing.T) {
	r := &fakeRunner{}
	m := actionModel(r, true, jump, action.Action{Name: "web", Key: "?", Run: "open", Detach: true})

	m, _ = press(m, question)

	if len(r.started) != 1 || strings.Contains(plain(m.View().Content), "navigation") {
		t.Errorf("started = %v, want the Action run and no overlay", r.started)
	}
}

func TestHelp_FooterPointsToItOnlyWhereItOpens(t *testing.T) {
	if line := footerOf(sizedActions(120, true, action.Builtins()...)); !strings.Contains(line, "? help") {
		t.Errorf("vim footer = %q, want the ? help pointer", line)
	}
	if line := footerOf(sizedActions(120, false, action.Builtins()...)); strings.Contains(line, "? help") {
		t.Errorf("default footer = %q, want no pointer without an overlay", line)
	}
}

func TestHelp_FitsTheTerminalHeightAndKeepsTheCloseLine(t *testing.T) {
	for _, height := range []int{40, 20, 9, 3} {
		m := picker.NewModel(manyRows(5), noopStatus, picker.Options{Vim: true, Actions: action.Builtins()})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: height})
		next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
		next, _ = next.(picker.Model).Update(question)

		lines := strings.Split(next.(picker.Model).View().Content, "\n")
		if len(lines) != height {
			t.Errorf("overlay at height %d is %d lines", height, len(lines))
		}
		if !strings.Contains(plain(lines[0]), "? closes this help") {
			t.Errorf("height %d: first line = %q, want the close hint", height, lines[0])
		}
		for _, l := range lines {
			if w := lipgloss.Width(l); w > 60 {
				t.Errorf("height %d: line %q is %d wide", height, l, w)
			}
		}
	}
}
