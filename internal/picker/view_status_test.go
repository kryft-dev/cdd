package picker_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kryft-dev/cdd/internal/git"
	"github.com/kryft-dev/cdd/internal/picker"
)

// statusView renders a one-row Picker whose status has landed as st, wide
// enough for the preview and short enough to drop the legend, so any "!" in
// the output comes from the row or the preview.
func statusView(t *testing.T, st git.Status) string {
	t.Helper()
	rows := []picker.Row{{Project: picker.Project{Dir: "~/work/", Name: "alpha", Path: "/root/work/alpha"}}}
	m := picker.NewModel(rows, keyedStatus(map[string]git.Status{"/root/work/alpha": st}), picker.Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 10})
	m = next.(picker.Model)
	for _, c := range m.Init()().(tea.BatchMsg) {
		next, _ = m.Update(c())
		m = next.(picker.Model)
	}
	return plain(m.View().Content)
}

// TestView_NotRepoIsNeutral verifies that a Project with no .git shows no
// "!" and a plain "not a git repo" in the preview, with no branch or sync.
func TestView_NotRepoIsNeutral(t *testing.T) {
	out := statusView(t, git.Status{Kind: git.NotRepo})
	if strings.Contains(out, "!") {
		t.Errorf("view has a !, want none for a non-git Project:\n%s", out)
	}
	if !strings.Contains(out, "not a git repo") {
		t.Errorf("view lacks %q:\n%s", "not a git repo", out)
	}
	for _, word := range []string{"branch", "sync"} {
		if strings.Contains(out, word) {
			t.Errorf("view has %q, want it omitted for a non-git Project:\n%s", word, out)
		}
	}
}

// TestView_UnknownStaysRed verifies that a real git failure keeps the "!"
// in the row and its words in the preview.
func TestView_UnknownStaysRed(t *testing.T) {
	out := statusView(t, git.Status{Kind: git.Unknown})
	if !strings.Contains(out, "!") || !strings.Contains(out, "git could not read it") {
		t.Errorf("view lacks the ! and %q for an Unknown status:\n%s", "git could not read it", out)
	}
}
