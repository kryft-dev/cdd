package picker_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/picker"
)

var ctrlY = tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}

// copyModel is an actionModel over the built-in copy Action, with copy as
// the clipboard function.
func copyModel(copyFn func(string) (bool, error)) picker.Model {
	m := twoRowModel(picker.Options{
		Actions: action.Builtins(),
		Runner:  &fakeRunner{},
		Copy:    copyFn,
	})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	return next.(picker.Model)
}

func TestModel_Copy_UsesTheClipboardProgramAndStaysOpen(t *testing.T) {
	var got []string
	m := copyModel(func(text string) (bool, error) {
		got = append(got, text)
		return true, nil
	})

	m, cmd := press(m, ctrlY)

	if len(got) != 1 || got[0] != "/root/work/alpha" {
		t.Errorf("copied %v, want the path of alpha", got)
	}
	if _, chosen := m.Chosen(); chosen || cmd != nil {
		t.Errorf("chosen = %v, Cmd = %v, want the Picker open and no OSC 52", chosen, cmd != nil)
	}
	if v := plain(m.View().Content); !strings.Contains(v, "copied /root/work/alpha") {
		t.Errorf("View lacks the copied message:\n%s", v)
	}
}

func TestModel_Copy_WithoutAProgramFallsBackToOSC52(t *testing.T) {
	m := copyModel(func(string) (bool, error) { return false, nil })

	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m, cmd := press(m, ctrlY)

	if cmd == nil {
		t.Fatal("Cmd = nil, want the OSC 52 clipboard command")
	}
	msg := cmd()
	if kind := fmt.Sprintf("%T", msg); kind != "tea.setClipboardMsg" {
		t.Errorf("Cmd gave %s, want tea.setClipboardMsg", kind)
	}
	if text := fmt.Sprintf("%s", msg); text != "/root/work/beta" {
		t.Errorf("OSC 52 text = %q, want the path of beta", text)
	}
	if v := plain(m.View().Content); !strings.Contains(v, "copied /root/work/beta") {
		t.Errorf("View lacks the copied message:\n%s", v)
	}
}

func TestModel_Copy_FailureShowsOnTheFooter(t *testing.T) {
	m := copyModel(func(string) (bool, error) { return true, errors.New("exit status 1") })

	m, cmd := press(m, ctrlY)

	if v := plain(m.View().Content); !strings.Contains(v, "copy: exit status 1") || strings.Contains(v, "copied") {
		t.Errorf("View lacks the failure:\n%s", v)
	}
	if cmd != nil {
		t.Error("Cmd != nil, want no OSC 52 after a failed program")
	}
}

func TestModel_Copy_MessageClearsOnTheNextKey(t *testing.T) {
	m := copyModel(func(string) (bool, error) { return true, nil })

	m, _ = press(m, ctrlY)
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})

	if v := plain(m.View().Content); strings.Contains(v, "copied") {
		t.Errorf("View still says copied after a key press:\n%s", v)
	}
}
