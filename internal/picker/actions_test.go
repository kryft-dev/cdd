package picker_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/picker"
)

// fakeRunner records the Actions Start is asked to run, and fails them
// with err.
type fakeRunner struct {
	started []string // "name path"
	err     error
}

func (f *fakeRunner) Start(a action.Action, path string) error {
	f.started = append(f.started, a.Name+" "+path)
	return f.err
}

func (f *fakeRunner) Run(action.Action, string) (int, error) {
	panic("the Picker never runs an Action in the foreground")
}

// actionModel is a sized Model over twoRowModel's rows with actions bound.
func actionModel(r action.Runner, vim bool, actions ...action.Action) picker.Model {
	m := twoRowModel(picker.Options{Vim: vim, Actions: actions, Runner: r})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	return next.(picker.Model)
}

func press(m picker.Model, msg tea.KeyPressMsg) (picker.Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(picker.Model), cmd
}

// jump is the built-in Jump, bound to enter.
var jump = action.Builtins()[0]

var (
	ctrlV = tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl}
	ctrlL = tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}
)

func TestModel_Action_AttachedQuitsWithTheActionAndRow(t *testing.T) {
	lazygit := action.Action{Name: "lazygit", Key: "ctrl+l", Run: "lazygit"}
	m := actionModel(&fakeRunner{}, false, lazygit)

	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, cmd := press(m, ctrlL)

	row, ok := m.Chosen()
	if !ok || row.Project.Name != "beta" {
		t.Errorf("Chosen = %v, %v, want beta", row.Project.Name, ok)
	}
	if got := m.ChosenAction(); got == nil || got.Name != "lazygit" {
		t.Errorf("ChosenAction = %v, want lazygit", got)
	}
	if cmd == nil {
		t.Error("Cmd = nil, want tea.Quit")
	}
}

func TestModel_Action_DetachedStartsAndKeepsThePickerOpen(t *testing.T) {
	r := &fakeRunner{}
	m := actionModel(r, false, action.Action{Name: "code", Key: "ctrl+v", Run: "code {path}", Detach: true})

	m, cmd := press(m, ctrlV)

	if len(r.started) != 1 || r.started[0] != "code /root/work/alpha" {
		t.Errorf("started = %v, want code on alpha", r.started)
	}
	if _, ok := m.Chosen(); ok || cmd != nil {
		t.Errorf("Chosen ok = %v, Cmd = %v, want the Picker still open", ok, cmd != nil)
	}
}

func TestModel_Action_DetachedFailureShowsOnTheFooterUntilTheNextKey(t *testing.T) {
	r := &fakeRunner{err: errors.New("no such directory")}
	m := actionModel(r, false, action.Action{Name: "code", Key: "ctrl+v", Run: "code", Detach: true})

	m, _ = press(m, ctrlV)
	if v := plain(m.View().Content); !strings.Contains(v, "code: no such directory") {
		t.Errorf("View lacks the failure:\n%s", v)
	}

	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if v := plain(m.View().Content); strings.Contains(v, "no such directory") {
		t.Errorf("View still shows the failure after a key press:\n%s", v)
	}
}

func TestModel_Action_KeyOverridesANavigationKey(t *testing.T) {
	r := &fakeRunner{}
	m := actionModel(r, false, jump, action.Action{Name: "code", Key: "ctrl+n", Run: "code", Detach: true})

	m, _ = press(m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter}) // the cursor did not move

	if len(r.started) != 1 {
		t.Errorf("started = %v, want the Action to run on ctrl+n", r.started)
	}
	if row, _ := m.Chosen(); row.Project.Name != "alpha" {
		t.Errorf("Chosen = %q, want alpha (ctrl+n no longer moves down)", row.Project.Name)
	}
}

func TestModel_Action_ArrowKeysStillMoveWhenCtrlNIsTaken(t *testing.T) {
	m := actionModel(&fakeRunner{}, false, jump, action.Action{Name: "code", Key: "ctrl+n", Run: "code", Detach: true})

	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if row, _ := m.Chosen(); row.Project.Name != "beta" {
		t.Errorf("Chosen = %q, want beta", row.Project.Name)
	}
}

func TestModel_Action_PlainKeyFiresOnlyInVimListFocus(t *testing.T) {
	r := &fakeRunner{}
	help := action.Action{Name: "web", Key: "w", Run: "open", Detach: true}
	m := actionModel(r, true, help)

	m, _ = press(m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	if len(r.started) != 1 {
		t.Fatalf("started = %v, want w to run in list focus", r.started)
	}

	m, _ = press(m, tea.KeyPressMsg{Code: 'f', Text: "f"}) // focus the filter
	m, _ = press(m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	if len(r.started) != 1 {
		t.Errorf("started = %v, want w to type in the filter focus", r.started)
	}
	if v := plain(m.View().Content); !strings.Contains(v, "❯ w") {
		t.Errorf("filter did not take the w:\n%s", v)
	}
}

var (
	enter    = tea.KeyPressMsg{Code: tea.KeyEnter}
	altEnter = tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt}
)

func TestModel_Jump_EnterJumpsInBothKeyMapsAndBothVimFocuses(t *testing.T) {
	tests := []struct {
		name string
		vim  bool
		keys []tea.KeyPressMsg
	}{
		{"default", false, nil},
		{"vim list focus", true, nil},
		{"vim filter focus", true, []tea.KeyPressMsg{{Code: 'f', Text: "f"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := actionModel(&fakeRunner{}, tt.vim, action.Builtins()...)
			for _, k := range tt.keys {
				m, _ = press(m, k)
			}

			m, cmd := press(m, enter)

			row, ok := m.Chosen()
			if !ok || row.Project.Name != "alpha" || cmd == nil {
				t.Errorf("Chosen = %q, %v, Cmd = %v, want alpha and a quit", row.Project.Name, ok, cmd != nil)
			}
			if got := m.ChosenAction(); got == nil || got.Name != "jump" || !got.Jump {
				t.Errorf("ChosenAction = %v, want jump", got)
			}
		})
	}
}

func TestModel_Jump_EnterDoesNothingWhenNoActionIsBoundToIt(t *testing.T) {
	m := actionModel(&fakeRunner{}, false)

	m, cmd := press(m, enter)

	if _, ok := m.Chosen(); ok || cmd != nil {
		t.Errorf("Chosen ok = %v, Cmd = %v, want enter to be unbound", ok, cmd != nil)
	}
}

func TestModel_Jump_MovedToAnotherKeyJumpsThere(t *testing.T) {
	r := &fakeRunner{}
	code := action.Action{Name: "code", Key: "enter", Run: "code {path}", Detach: true}
	moved := action.Action{Name: "jump", Key: "alt+enter", Jump: true}
	m := actionModel(r, false, moved, code)

	m, _ = press(m, enter)
	if len(r.started) != 1 || r.started[0] != "code /root/work/alpha" {
		t.Fatalf("started = %v, want enter to run code", r.started)
	}
	if _, ok := m.Chosen(); ok {
		t.Fatal("enter Jumped although code owns it")
	}

	m, _ = press(m, altEnter)
	if got := m.ChosenAction(); got == nil || got.Name != "jump" {
		t.Errorf("ChosenAction = %v, want jump on alt+enter", got)
	}
}

func TestModel_Action_NoRowsIsANoOp(t *testing.T) {
	r := &fakeRunner{}
	m := actionModel(r, false, action.Action{Name: "code", Key: "ctrl+v", Run: "code", Detach: true})
	for _, c := range "zzz" {
		m, _ = press(m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}

	_, cmd := press(m, ctrlV)
	if len(r.started) != 0 || cmd != nil {
		t.Errorf("started = %v, Cmd = %v, want nothing", r.started, cmd != nil)
	}
}

func TestModel_Action_RemoteWithoutOneShowsTheErrorAndStaysOpen(t *testing.T) {
	r := &fakeRunner{err: git.ErrNoRemote}
	m := actionModel(r, false, action.Action{Name: "remote", Key: "ctrl+g", Run: "open {remote}", Detach: true})

	m, cmd := press(m, tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	if v := plain(m.View().Content); !strings.Contains(v, "remote: no git remote") {
		t.Errorf("View lacks the error:\n%s", v)
	}
	if _, chosen := m.Chosen(); chosen || cmd != nil {
		t.Errorf("the Picker left (chosen = %v, Cmd = %v), want it open", chosen, cmd != nil)
	}
}
