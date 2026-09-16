// PROTOTYPE: throwaway Bubble Tea mock of the cdd Picker with fake data.
//
// Three structurally different variants of the Picker, switchable with Tab
// (or -variant a|b|c). The magenta bar at the bottom is the prototype
// switcher, not part of the design under review.
//
//	go run ./prototype/picker              # variant A, theme from terminal
//	go run ./prototype/picker -variant c   # start on variant C
//	go run ./prototype/picker -theme light # force the light palette
//	go run ./prototype/picker -snapshot    # print every variant, no TTY needed
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

type variant int

const (
	vTable variant = iota
	vGrouped
	vSplit
	vComposite
	variantCount
)

var variantNames = map[variant]string{
	vTable:     "A · Table (gh-dash)",
	vGrouped:   "B · Grouped by Kind",
	vSplit:     "C · List + preview",
	vComposite: "D · Grouped + preview",
}

// row is a visible Picker row: the Project plus its fuzzy match positions
// into Key() ("kind/name"), empty when no filter is active.
type row struct {
	p       *project
	matched []int
}

type model struct {
	projects  []project
	query     string
	cursor    int
	width     int
	height    int
	variant   variant
	themeMode string // auto | dark | light
	termDark  bool
	chosen    string
	keymap    string // arrows | vim
	filtering bool   // vim mode only: true while the filter has focus
}

func newModel(v variant, themeMode, keymap string) model {
	return model{projects: fakeProjects(), variant: v, themeMode: themeMode, termDark: true, width: 100, height: 30, keymap: keymap, filtering: keymap != "vim"}
}

func (m model) theme() theme {
	switch m.themeMode {
	case "dark":
		return newTheme(true)
	case "light":
		return newTheme(false)
	}
	return newTheme(m.termDark)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, statusCmds(m.projects))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.termDark = msg.IsDark()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case statusMsg:
		for i := range m.projects {
			if m.projects[i].Key() == msg.key {
				m.projects[i].Loaded = true
			}
		}
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		if rows := m.visible(); len(rows) > 0 {
			m.chosen = rows[m.cursor].p.Path()
		}
		return m, tea.Quit
	case "tab":
		m.variant = (m.variant + 1) % variantCount
	case "shift+tab":
		m.variant = (m.variant + variantCount - 1) % variantCount
	case "ctrl+t":
		m.themeMode = map[string]string{"auto": "dark", "dark": "light", "light": "auto"}[m.themeMode]
	case "ctrl+r":
		for i := range m.projects {
			m.projects[i].Loaded = false
		}
		return m, statusCmds(m.projects)
	case "ctrl+f":
		m.keymap = map[string]string{"arrows": "vim", "vim": "arrows"}[m.keymap]
		m.filtering = m.keymap != "vim"
	case "up", "ctrl+p", "ctrl+k":
		m.cursor = max(m.cursor-1, 0)
	case "down", "ctrl+n", "ctrl+j":
		m.cursor = min(m.cursor+1, max(len(m.visible())-1, 0))
	}
	if m.keymap == "vim" && !m.filtering {
		return m.vimKey(msg)
	}
	switch msg.String() {
	case "esc":
		if m.keymap == "vim" {
			m.filtering = false
			return m, nil
		}
		return m, tea.Quit
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
		m.cursor = 0
	case "ctrl+u":
		m.query, m.cursor = "", 0
	default:
		if msg.Text != "" {
			m.query += msg.Text
			m.cursor = 0
		}
	}
	return m, nil
}

// vimKey handles normal mode when -keys vim: the list has focus, j/k move,
// f hands focus to the filter, esc cancels the Picker.
func (m model) vimKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m, tea.Quit
	case "j":
		m.cursor = min(m.cursor+1, max(len(m.visible())-1, 0))
	case "k":
		m.cursor = max(m.cursor-1, 0)
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = max(len(m.visible())-1, 0)
	case "f", "/":
		m.filtering = true
	case "ctrl+u":
		m.query, m.cursor = "", 0
	}
	return m, nil
}

// visible applies the fuzzy filter over "kind/name". Empty query keeps
// History order; otherwise best match first.
func (m model) visible() []row {
	if m.query == "" {
		out := make([]row, len(m.projects))
		for i := range m.projects {
			out[i] = row{p: &m.projects[i]}
		}
		return out
	}
	keys := make([]string, len(m.projects))
	for i, p := range m.projects {
		keys[i] = p.Key()
	}
	var out []row
	for _, mt := range fuzzy.Find(m.query, keys) {
		out = append(out, row{p: &m.projects[mt.Index], matched: mt.MatchedIndexes})
	}
	return out
}

func (m model) View() tea.View {
	t := m.theme()
	body := m.render(t, m.width, m.height-1)
	v := tea.NewView(body + "\n" + m.switcher(t))
	v.AltScreen = true
	return v
}

func (m model) render(t theme, w, h int) string {
	switch m.variant {
	case vGrouped:
		return viewGrouped(m, t, w, h)
	case vSplit:
		return viewSplit(m, t, w, h)
	case vComposite:
		return viewComposite(m, t, w, h)
	}
	return viewTable(m, t, w, h)
}

// switcher is the prototype-only bar: deliberately garish so nobody mistakes
// it for part of the Picker.
func (m model) switcher(t theme) string {
	bar := lipgloss.NewStyle().Background(t.barBg).Foreground(t.barFg)
	mode := m.themeMode
	if mode == "auto" {
		mode = map[bool]string{true: "auto→dark", false: "auto→light"}[m.termDark]
	}
	s := fmt.Sprintf(" PROTOTYPE  ◀ shift+tab  %s  tab ▶   ctrl+t theme: %s   ctrl+f keys: %s   ctrl+r reload ", variantNames[m.variant], mode, m.keymap)
	return bar.Width(m.width).Render(s)
}

// filterLine is shared chrome: the prompt with a block cursor.
func (m model) filterLine(t theme, prompt string) string {
	cur := lipgloss.NewStyle().Reverse(true).Render(" ")
	if m.keymap == "vim" && !m.filtering {
		hint := t.Muted().Render("  f to filter")
		if m.query == "" {
			return t.Muted().Render(prompt) + hint
		}
		return t.Muted().Render(prompt) + m.query + hint
	}
	if m.query == "" {
		return t.fg(t.purple).Bold(true).Render(prompt) + cur + t.Muted().Render(" type to filter")
	}
	return t.fg(t.purple).Bold(true).Render(prompt) + m.query + cur
}

func main() {
	v := flag.String("variant", "d", "starting variant: a, b, c or d")
	th := flag.String("theme", "auto", "palette: auto, dark or light")
	keys := flag.String("keys", "arrows", "key map: arrows or vim (j/k move, f filters)")
	snap := flag.Bool("snapshot", false, "print every variant in both themes and exit")
	flag.Parse()
	start := variant(strings.Index("abcd", strings.ToLower(*v)))
	if start < 0 {
		start = vTable
	}
	m := newModel(start, *th, *keys)
	if *snap {
		snapshot(m)
		return
	}
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if fm := final.(model); fm.chosen != "" {
		fmt.Println(fm.chosen)
	} else {
		os.Exit(130)
	}
}

// snapshot renders all variants with every status loaded, for review
// without a terminal (pipe through `less -R` or strip the ANSI).
func snapshot(m model) {
	for i := range m.projects {
		m.projects[i].Loaded = true
	}
	m.cursor = 1
	m.filtering = true
	for _, mode := range []string{"dark", "light"} {
		m.themeMode = mode
		for v := vTable; v < variantCount; v++ {
			m.variant = v
			fmt.Printf("═══ %s · %s ═══\n%s\n%s\n\n", variantNames[v], mode, m.render(m.theme(), m.width, m.height-1), m.switcher(m.theme()))
		}
	}
}
