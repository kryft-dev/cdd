package main

// PROTOTYPE: throwaway fake data for the Picker look mock. Not production code.

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const fakeRoot = "~/Projects"

// project is one Picker row. Loaded is false until the fake async status
// arrives, mirroring the real one-tea.Cmd-per-row pattern.
type project struct {
	Kind, Name, Branch string
	Repo               bool
	Loaded             bool
	Dirty, Untracked   bool
	Ahead, Behind      int
	Upstream           bool
	Timeout            bool
	Last               time.Time // zero means never visited
	Visits             int
}

func (p project) Key() string  { return p.Kind + "/" + p.Name }
func (p project) Path() string { return fakeRoot + "/" + p.Key() }

var now = time.Date(2026, 9, 16, 21, 30, 0, 0, time.Local)

func ago(d time.Duration) time.Time { return now.Add(-d) }

// fakeProjects returns Projects already in History order: most recent Visit
// first, never-visited last and alphabetical.
func fakeProjects() []project {
	ps := []project{
		{Kind: "tools", Name: "cdd", Branch: "main", Repo: true, Dirty: true, Ahead: 1, Upstream: true, Last: ago(12 * time.Minute), Visits: 41},
		{Kind: "work", Name: "api-gateway", Branch: "feat/rate-limits", Repo: true, Dirty: true, Untracked: true, Ahead: 3, Behind: 2, Upstream: true, Last: ago(2 * time.Hour), Visits: 128},
		{Kind: "work", Name: "billing", Branch: "main", Repo: true, Upstream: true, Last: ago(5 * time.Hour), Visits: 77},
		{Kind: "oss", Name: "bubbletea", Branch: "v2-fixes", Repo: true, Behind: 14, Upstream: true, Last: ago(26 * time.Hour), Visits: 9},
		{Kind: "work", Name: "design-system", Branch: "main", Repo: true, Untracked: true, Upstream: true, Last: ago(2 * 24 * time.Hour), Visits: 33},
		{Kind: "play", Name: "raytracer", Branch: "master", Repo: true, Dirty: true, Last: ago(3 * 24 * time.Hour), Visits: 5},
		{Kind: "courses", Name: "distributed-systems", Repo: false, Last: ago(6 * 24 * time.Hour), Visits: 12},
		{Kind: "tools", Name: "dotfiles", Branch: "main", Repo: true, Upstream: true, Last: ago(9 * 24 * time.Hour), Visits: 60},
		{Kind: "oss", Name: "lipgloss", Branch: "main", Repo: true, Upstream: true, Last: ago(15 * 24 * time.Hour), Visits: 3},
		{Kind: "work", Name: "mobile-app", Branch: "release/3.2", Repo: true, Timeout: true, Last: ago(20 * 24 * time.Hour), Visits: 18},
		{Kind: "play", Name: "advent-of-code", Branch: "main", Repo: true, Ahead: 2, Upstream: true, Last: ago(45 * 24 * time.Hour), Visits: 22},
		{Kind: "courses", Name: "rust-book", Branch: "main", Repo: true, Upstream: true, Last: ago(70 * 24 * time.Hour), Visits: 7},
		{Kind: "oss", Name: "zoxide", Branch: "main", Repo: true, Behind: 3, Upstream: true, Last: ago(200 * 24 * time.Hour), Visits: 2},
		{Kind: "work", Name: "legacy-crm", Branch: "trunk", Repo: true, Dirty: true, Last: ago(400 * 24 * time.Hour), Visits: 4},
		// Never visited: History has nothing for these, so they sort last, A-Z.
		{Kind: "play", Name: "chip8", Branch: "main", Repo: true, Upstream: true},
		{Kind: "tools", Name: "scripts", Repo: false},
		{Kind: "work", Name: "wiki", Branch: "main", Repo: true, Untracked: true, Upstream: true},
	}
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if a.Last.IsZero() != b.Last.IsZero() {
			return !a.Last.IsZero()
		}
		if a.Last.IsZero() {
			return a.Key() < b.Key()
		}
		return a.Last.After(b.Last)
	})
	return ps
}

// statusMsg is the fake per-row git status result, keyed by Project key.
type statusMsg struct{ key string }

// statusCmds simulates one async git status per row landing at random times.
func statusCmds(ps []project) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(ps))
	for _, p := range ps {
		key := p.Key()
		d := 150*time.Millisecond + time.Duration(rand.Intn(2200))*time.Millisecond
		cmds = append(cmds, tea.Tick(d, func(time.Time) tea.Msg { return statusMsg{key: key} }))
	}
	return tea.Batch(cmds...)
}

// relTime renders a Visit time relative to now; empty for never visited.
func relTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/24/7))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

func absTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("Mon 2 Jan 2006 15:04")
}

func plural(n int, s string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, s)
	}
	return fmt.Sprintf("%d %ss", n, s)
}

func pad(s string, w int) string {
	if n := w - len([]rune(s)); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}
