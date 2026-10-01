package picker_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/picker"
)

var (
	ctrlD = tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	keyY  = tea.KeyPressMsg{Code: 'y', Text: "y"}
	keyN  = tea.KeyPressMsg{Code: 'n', Text: "n"}
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
)

// forgetAction is the built-in forget Action, bound to ctrl+d.
var forgetAction = action.Action{Name: "forget", Key: "ctrl+d", Internal: action.InternalForget}

// forgetModel is a sized Model over alpha, beta and gamma with the Jump and
// forget Actions bound, and forget as the Store's Forget. It returns the
// paths forget was asked for.
func forgetModel(vim bool, forget func(string) error) (picker.Model, *[]string) {
	var asked []string
	rows := []picker.Row{
		{Project: picker.Project{Dir: "~/work/", Name: "alpha", Path: "/root/work/alpha"}},
		{Project: picker.Project{Dir: "~/work/", Name: "beta", Path: "/root/work/beta"}},
		{Project: picker.Project{Dir: "~/work/", Name: "gamma", Path: "/root/work/gamma"}},
	}
	m := picker.NewModel(rows, noopStatus, picker.Options{
		Vim:     vim,
		Actions: []action.Action{jump, forgetAction},
		Forget: func(path string) error {
			asked = append(asked, path)
			if forget == nil {
				return nil
			}
			return forget(path)
		},
	})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	return next.(picker.Model), &asked
}

// aboveFooter is the View without its last line, where a message that names
// a Project would otherwise look like a row.
func aboveFooter(m picker.Model) string {
	lines := strings.Split(plain(m.View().Content), "\n")
	return strings.Join(lines[:len(lines)-1], "\n")
}

// selected is the name of the row under the cursor, found by choosing it.
func selected(t *testing.T, m picker.Model) string {
	t.Helper()
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	row, ok := m.Chosen()
	if !ok {
		t.Fatal("enter chose nothing")
	}
	return row.Project.Name
}

func TestModel_Forget_AsksOnTheFooterAndKeepsTheRowUntilConfirmed(t *testing.T) {
	m, asked := forgetModel(false, nil)

	m, _ = press(m, down)
	m, cmd := press(m, ctrlD)

	if line := footerOf(m); !strings.HasPrefix(line, "forget beta? y/n") {
		t.Errorf("footer = %q, want the prompt in place of the hints", line)
	}
	if len(*asked) != 0 || cmd != nil {
		t.Errorf("forgot %v, Cmd = %v, want nothing before the answer", *asked, cmd != nil)
	}
	if v := plain(m.View().Content); !strings.Contains(v, "beta") {
		t.Errorf("View lost the row before the answer:\n%s", v)
	}
}

func TestModel_Forget_YForgetsAndDropsTheRow(t *testing.T) {
	m, asked := forgetModel(false, nil)

	m, _ = press(m, down)
	m, _ = press(m, ctrlD)
	m, _ = press(m, keyY)

	if len(*asked) != 1 || (*asked)[0] != "/root/work/beta" {
		t.Errorf("forgot %v, want the path of beta", *asked)
	}
	if v := aboveFooter(m); strings.Contains(v, "beta") || !strings.Contains(v, "alpha") || !strings.Contains(v, "gamma") {
		t.Errorf("View still lists beta, or lost another row:\n%s", v)
	}
	if line := footerOf(m); !strings.HasPrefix(line, "forgot beta") || !strings.HasSuffix(line, "2/2") {
		t.Errorf("footer = %q, want the confirmation and the new count", line)
	}
}

func TestModel_Forget_AnyOtherKeyCancelsAndIsConsumed(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{keyN, {Code: tea.KeyEscape}, {Code: tea.KeyEnter}, {Code: 'c', Mod: tea.ModCtrl}, down, {Code: 'x', Text: "x"}} {
		m, asked := forgetModel(false, nil)

		m, _ = press(m, ctrlD)
		m, cmd := press(m, key)

		if len(*asked) != 0 {
			t.Errorf("%v: forgot %v, want nothing", key, *asked)
		}
		if _, chosen := m.Chosen(); chosen || cmd != nil {
			t.Errorf("%v: Chosen = %v, Cmd = %v, want the key consumed", key, chosen, cmd != nil)
		}
		if line := footerOf(m); !strings.HasPrefix(line, "enter jump") {
			t.Errorf("%v: footer = %q, want the hints back", key, line)
		}
		if got := selected(t, m); got != "alpha" {
			t.Errorf("%v: cursor on %q, want it unmoved on alpha", key, got)
		}
	}
}

func TestModel_Forget_TheAnswerIsNotTypedIntoTheFilter(t *testing.T) {
	for _, vim := range []bool{false, true} {
		m, _ := forgetModel(vim, nil)
		if vim {
			m, _ = press(m, tea.KeyPressMsg{Code: 'f', Text: "f"})
		}

		m, _ = press(m, ctrlD)
		m, _ = press(m, keyN)
		m, _ = press(m, ctrlD)
		m, _ = press(m, keyY)

		if v := plain(m.View().Content); !strings.Contains(v, "alpha") || !strings.Contains(v, "beta") {
			t.Errorf("vim=%v: the filter ate an answer:\n%s", vim, v)
		}
	}
}

func TestModel_Forget_CursorStaysOnTheNeighbouringRow(t *testing.T) {
	tests := []struct {
		name  string
		moves int
		want  string // the row under the cursor afterwards
	}{
		{"first row, the next one takes its place", 0, "beta"},
		{"middle row, the one below takes its place", 1, "gamma"},
		{"last row, the cursor steps up", 2, "beta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := forgetModel(false, nil)
			for range tt.moves {
				m, _ = press(m, down)
			}

			m, _ = press(m, ctrlD)
			m, _ = press(m, keyY)

			if got := selected(t, m); got != tt.want {
				t.Errorf("cursor on %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModel_Forget_TheOnlyRowLeavesAnEmptyList(t *testing.T) {
	var asked []string
	rows := []picker.Row{{Project: picker.Project{Dir: "~/work/", Name: "alpha", Path: "/root/work/alpha"}}}
	m := picker.NewModel(rows, noopStatus, picker.Options{
		Actions: []action.Action{jump, forgetAction},
		Forget:  func(path string) error { asked = append(asked, path); return nil },
	})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	m = next.(picker.Model)

	m, _ = press(m, ctrlD)
	m, _ = press(m, keyY)

	if len(asked) != 1 {
		t.Errorf("forgot %v, want alpha", asked)
	}
	if v := plain(m.View().Content); strings.Contains(v, "alpha") {
		t.Errorf("View still lists alpha:\n%s", v)
	}
	// With no row left, forgetting again does nothing.
	m, _ = press(m, ctrlD)
	if v := plain(m.View().Content); strings.Contains(v, "forget") {
		t.Errorf("View = %q, want no prompt with no row", v)
	}
}

func TestModel_Forget_UnderAFilterDropsTheRowAmongTheVisibleOnes(t *testing.T) {
	m, asked := forgetModel(false, nil)
	m, _ = press(m, tea.KeyPressMsg{Code: 'a', Text: "a"})
	m, _ = press(m, tea.KeyPressMsg{Code: 'm', Text: "m"})

	m, _ = press(m, ctrlD)
	m, _ = press(m, keyY)

	if len(*asked) != 1 || (*asked)[0] != "/root/work/gamma" {
		t.Errorf("forgot %v, want the path of gamma, the best match for \"am\"", *asked)
	}
	if v := aboveFooter(m); strings.Contains(v, "gamma") {
		t.Errorf("View still lists gamma:\n%s", v)
	}
}

func TestModel_Forget_FailureShowsOnTheFooterAndKeepsTheRow(t *testing.T) {
	m, _ := forgetModel(false, func(string) error { return errors.New("disk full") })

	m, _ = press(m, down)
	m, _ = press(m, ctrlD)
	m, cmd := press(m, keyY)

	if line := footerOf(m); !strings.HasPrefix(line, "forget: disk full") || !strings.HasSuffix(line, "3/3") {
		t.Errorf("footer = %q, want the error and the unchanged count", line)
	}
	if v := plain(m.View().Content); !strings.Contains(v, "beta") {
		t.Errorf("View lost beta although forgetting failed:\n%s", v)
	}
	if got := selected(t, m); got != "beta" {
		t.Errorf("cursor on %q, want it kept on beta", got)
	}
	if cmd != nil {
		t.Error("Cmd != nil, want the Picker left open")
	}
}

func TestModel_Forget_WithoutAForgetFuncAsksNothing(t *testing.T) {
	m := actionModel(&fakeRunner{}, false, jump, forgetAction)

	m, _ = press(m, ctrlD)

	if line := footerOf(m); strings.Contains(line, "forget alpha?") || !strings.HasPrefix(line, "enter jump") {
		t.Errorf("footer = %q, want the hints and no prompt that cannot be honoured", line)
	}
}

func TestModel_Forget_TheVimHelpOverlayDescribesIt(t *testing.T) {
	m, _ := forgetModel(true, nil)

	m, _ = press(m, tea.KeyPressMsg{Code: '?', Text: "?"})

	if v := plain(m.View().Content); !strings.Contains(v, "ctrl+d") || !strings.Contains(v, "forgets the Project") {
		t.Errorf("help overlay lacks forget:\n%s", v)
	}
}

func TestModel_Forget_ReboundKeyAsksToo(t *testing.T) {
	moved := forgetAction
	moved.Key = "ctrl+k"
	m := twoRowModel(picker.Options{Actions: []action.Action{jump, moved}, Forget: func(string) error { return nil }})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	m = next.(picker.Model)

	m, _ = press(m, tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})

	if line := footerOf(m); !strings.HasPrefix(line, "forget alpha? y/n") {
		t.Errorf("footer = %q, want the prompt on the rebound key", line)
	}
}
