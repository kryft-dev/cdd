package picker

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kryft-dev/cdd/internal/git"
)

// paletteModel is a Model sized to a terminal but told nothing yet about
// its background colour: the state the Picker opens in.
func paletteModel() Model {
	rows := []Row{{Project: Project{Kind: "work", Name: "alpha", Path: "/root/work/alpha"}}}
	m := NewModel(rows, func(context.Context, string) git.Status { return git.Status{} }, Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	return next.(Model)
}

// TestModel_View_HoldsTheFrameUntilThePaletteSettles verifies that nothing
// is drawn before the terminal reports its background colour, and that the
// deadline releases the frame regardless. Drawing first and asking after
// meant the whole Picker repainted in the other palette a moment into its
// life, which is the flash a user sees on launch.
func TestModel_View_HoldsTheFrameUntilThePaletteSettles(t *testing.T) {
	m := paletteModel()
	if content := m.View().Content; content != "" {
		t.Errorf("View() before the background report drew %q, want nothing yet", content)
	}

	next, _ := m.Update(paletteDeadlineMsg{})
	if content := next.(Model).View().Content; content == "" {
		t.Errorf("View() after the deadline drew nothing; a silent terminal must not hold the frame")
	}
}

// TestModel_View_DeadlineDrawsTheDarkFrame pins the frame a silent
// terminal gets to the one a dark terminal gets: dark is the assumption
// lipgloss itself makes, and the one a terminal that will not answer is
// most likely to want.
func TestModel_View_DeadlineDrawsTheDarkFrame(t *testing.T) {
	timedOut, _ := paletteModel().Update(paletteDeadlineMsg{})
	reported, _ := paletteModel().Update(tea.BackgroundColorMsg{Color: lipgloss.Color("#0D1117")})

	if silent, dark := timedOut.(Model).View().Content, reported.(Model).View().Content; silent != dark {
		t.Errorf("the frame drawn on the deadline is not the dark frame:\ndeadline:\n%s\ndark:\n%s", silent, dark)
	}
}
