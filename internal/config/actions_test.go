package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/action"
	"github.com/kryft-dev/cdd/internal/config"
)

func TestLoadFromActions(t *testing.T) {
	cfg, err := config.LoadFrom(writeConfig(t, `
[actions.code]
key    = "ctrl+v"
run    = "code {path}"
detach = true

[actions.lazygit]
key  = "alt+ctrl+l"
run  = "lazygit"
jump = true
`))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	want := []action.Action{
		{Name: "code", Key: "ctrl+v", Run: "code {path}", Detach: true},
		{Name: "lazygit", Key: "ctrl+alt+l", Run: "lazygit", Jump: true},
	}
	got := cfg.ResolvedActions[:2]
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("action %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLoadFromMissingFileResolvesBuiltins(t *testing.T) {
	cfg, err := config.LoadFrom(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.ResolvedActions) != len(action.Builtins()) {
		t.Errorf("ResolvedActions = %v, want the built-ins", cfg.ResolvedActions)
	}
}

func TestLoadFromActionsRejected(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"unknown key name", "[actions.a]\nrun = \"x\"\nkey = \"ctrl+\"\n", "line 3"},
		{"esc", "[actions.a]\nrun = \"x\"\nkey = \"esc\"\n", "line 3"},
		{"ctrl+c", "[actions.a]\nkey = \"ctrl+c\"\nrun = \"x\"\n", "line 2"},
		{"printable key", "[actions.a]\nrun = \"x\"\nkey = \"a\"\n", "keys.vim"},
		{"missing run", "\n[actions.a]\nkey = \"ctrl+a\"\n", "line 2"},
		{
			"two Actions on a key",
			"[actions.a]\nrun = \"x\"\nkey = \"ctrl+a\"\n[actions.b]\nrun = \"y\"\nkey = \"ctrl+a\"\n",
			"line 6",
		},
		{"unknown field", "[actions.a]\nrun = \"x\"\nbogus = 1\n", "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.LoadFrom(writeConfig(t, tt.body))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadFrom error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestLoadFromPrintableActionKeyAllowedWithVim(t *testing.T) {
	cfg, err := config.LoadFrom(writeConfig(t, "[keys]\nvim = true\n[actions.help]\nrun = \"x\"\nkey = \"?\"\n"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	first := cfg.ResolvedActions[0]
	if first.Name != "help" || first.Key != "?" {
		t.Errorf("first action = %+v", first)
	}
}

func TestLoadFromActionsKeepConfigOrderBeforeTheBuiltins(t *testing.T) {
	cfg, err := config.LoadFrom(writeConfig(t, `
[actions.zed]
run = "zed {path}"

[actions.editor]
run = "hx {path}"

[actions."a b"]
run = "x"

[actions.code]
run = "code {path}"
`))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	var got []string
	for _, a := range cfg.ResolvedActions {
		got = append(got, a.Name)
	}
	want := []string{"zed", "a b", "code"}
	for _, a := range action.Builtins() {
		want = append(want, a.Name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}
