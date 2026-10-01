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

var errBoom = errors.New("boom")

// footerOf is a sized Model's last line: the keys line, with the match
// count on its right.
func footerOf(m picker.Model) string {
	lines := strings.Split(plain(m.View().Content), "\n")
	return lines[len(lines)-1]
}

// sizedActions is a Model with actions bound, sized width by 40.
func sizedActions(width int, vim bool, actions ...action.Action) picker.Model {
	m := twoRowModel(picker.Options{Vim: vim, Actions: actions})
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	return next.(picker.Model)
}

func TestFooter_WideListsEveryBoundActionBuiltinsInTheirOrder(t *testing.T) {
	m := sizedActions(120, false, action.Builtins()...)

	line := footerOf(m)

	const want = "enter jump · ctrl+o files · ctrl+e editor · ctrl+g remote · ctrl+y copy"
	if !strings.HasPrefix(line, want) {
		t.Errorf("footer = %q, want it to start %q", line, want)
	}
	if strings.Contains(line, "…") || !strings.HasSuffix(line, "2/2") {
		t.Errorf("footer = %q, want nothing dropped and the count at the right edge", line)
	}
	if w := lipgloss.Width(line); w != 120 {
		t.Errorf("footer is %d wide, want 120", w)
	}
}

func TestFooter_NarrowDropsWholeHintsBehindAnEllipsis(t *testing.T) {
	m := sizedActions(40, false, action.Builtins()...)

	line := footerOf(m)

	if !strings.HasPrefix(line, "enter jump · ctrl+o files …") {
		t.Errorf("footer = %q, want the first hints then an ellipsis", line)
	}
	if strings.Contains(line, "ctrl+e") {
		t.Errorf("footer = %q, want ctrl+e dropped whole", line)
	}
	if w := lipgloss.Width(line); w != 40 {
		t.Errorf("footer is %d wide, want 40", w)
	}
}

func TestFooter_NeverWiderThanTheTerminal(t *testing.T) {
	for width := 5; width <= 80; width++ {
		line := footerOf(sizedActions(width, false, action.Builtins()...))
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("footer at width %d is %d wide: %q", width, w, line)
		}
	}
}

func TestFooter_HintOrderIsEnterFirstThenTheOrderGiven(t *testing.T) {
	code := action.Action{Name: "code", Key: "enter", Run: "code {path}", Detach: true}
	lazy := action.Action{Name: "lazygit", Key: "ctrl+l", Run: "lazygit"}
	moved := action.Action{Name: "jump", Key: "alt+enter", Jump: true}
	unbound := action.Action{Name: "off", Key: "", Run: "x"}
	m := sizedActions(120, false, lazy, moved, unbound, code)

	line := footerOf(m)

	if want := "enter code · ctrl+l lazygit · alt+enter jump"; !strings.HasPrefix(line, want) {
		t.Errorf("footer = %q, want it to start %q", line, want)
	}
	if strings.Contains(line, "off") {
		t.Errorf("footer = %q lists the unbound Action", line)
	}
}

func TestFooter_NoActionsNoHints(t *testing.T) {
	line := footerOf(sizedActions(80, false))

	if got := strings.TrimSpace(line); got != "2/2" {
		t.Errorf("footer = %q, want only the count", line)
	}
}

func TestFooter_MessageReplacesHintsUntilTheNextKey(t *testing.T) {
	r := &fakeRunner{err: errBoom}
	web := action.Action{Name: "web", Key: "ctrl+w", Run: "open", Detach: true}
	m := actionModel(r, false, jump, web)

	m, _ = press(m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	line := footerOf(m)
	if !strings.HasPrefix(line, "web: ") || strings.Contains(line, "enter jump") || !strings.HasSuffix(line, "2/2") {
		t.Errorf("footer = %q, want the message in place of the hints, then the count", line)
	}

	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if line := footerOf(m); !strings.HasPrefix(line, "enter jump") {
		t.Errorf("footer = %q, want the hints back after a key press", line)
	}
}

func TestFooter_ErrorAndConfirmationAreDrawnInDifferentColours(t *testing.T) {
	m := actionModel(&fakeRunner{err: errBoom}, false, jump, action.Action{Name: "web", Key: "ctrl+w", Run: "open", Detach: true})
	bad, _ := press(m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})

	good, _ := press(copyModel(func(string) (bool, error) { return true, nil }), ctrlY)

	last := func(m picker.Model) string {
		lines := strings.Split(m.View().Content, "\n")
		return lines[len(lines)-1]
	}
	if last(bad)[:20] == last(good)[:20] {
		t.Errorf("error and confirmation share a style:\n%q\n%q", last(bad), last(good))
	}
}

func TestFooter_HideHintsKeepsTheCountAndMessages(t *testing.T) {
	r := &fakeRunner{err: errBoom}
	web := action.Action{Name: "web", Key: "ctrl+w", Run: "open", Detach: true}
	m := twoRowModel(picker.Options{Actions: []action.Action{jump, web}, Runner: r, HideHints: true})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	next, _ = next.(picker.Model).Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})
	m = next.(picker.Model)

	if got := strings.TrimSpace(footerOf(m)); got != "2/2" {
		t.Errorf("footer = %q, want only the count", got)
	}

	m, _ = press(m, tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	if line := footerOf(m); !strings.HasPrefix(line, "web: ") {
		t.Errorf("footer = %q, want the failure still shown", line)
	}
}
