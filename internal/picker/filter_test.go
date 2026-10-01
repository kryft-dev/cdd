package picker_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/picker"
)

// sameNameRows holds two Projects sharing a name under different parents,
// the newer first.
func sameNameRows() []picker.Row {
	return []picker.Row{
		{Project: picker.Project{Dir: "~/domain/foo.com/", Name: "barbar", Path: "/root/domain/foo.com/barbar"}},
		{Project: picker.Project{Dir: "~/domain/baz.com/", Name: "barbar", Path: "/root/domain/baz.com/barbar"}},
		{Project: picker.Project{Dir: "~/tools/", Name: "cdd", Path: "/root/tools/cdd"}},
	}
}

// typed returns m after each rune of s is typed into the filter, a space
// arriving as the space key does.
func typed(m picker.Model, s string) picker.Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(picker.Model)
	for _, r := range s {
		msg := tea.KeyPressMsg{Text: string(r)}
		if r == ' ' {
			msg.Code = tea.KeySpace
		}
		next, _ = m.Update(msg)
		m = next.(picker.Model)
	}
	return m
}

// chosen returns the Project path enter chooses on m, or "" for none.
func chosen(m picker.Model) string {
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	row, ok := next.(picker.Model).Chosen()
	if !ok {
		return ""
	}
	return row.Project.Path
}

func TestModel_FilterWords(t *testing.T) {
	tests := []struct {
		name, query, want string
	}{
		{"whitespace only keeps History order", " ", "/root/domain/foo.com/barbar"},
		{"parent then name", "baz br", "/root/domain/baz.com/barbar"},
		{"trailing space filters by parent", "baz ", "/root/domain/baz.com/barbar"},
		{"whole path without a space", "cdd", "/root/tools/cdd"},
		{"typo still finds the Project", "baz brabar", "/root/domain/baz.com/barbar"},
		{"no match chooses nothing", "zzz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := typed(picker.NewModel(sameNameRows(), noopStatus, picker.Options{Actions: action.Builtins()}), tt.query)
			if got := chosen(m); got != tt.want {
				t.Errorf("query %q chose %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

// TestModel_FilterTiesKeepHistoryOrder checks that equally good matches
// stay in History order, the newer first.
func TestModel_FilterTiesKeepHistoryOrder(t *testing.T) {
	m := typed(picker.NewModel(sameNameRows(), noopStatus, picker.Options{Actions: action.Builtins()}), " barbar")
	if got, want := chosen(m), "/root/domain/foo.com/barbar"; got != want {
		t.Errorf("chose %q, want %q", got, want)
	}
}
