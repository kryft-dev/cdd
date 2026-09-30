package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/config"
)

// writeConfig writes body to a config.toml under a fresh temp directory and
// returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// assertDefaults checks every key of cfg holds its default.
func assertDefaults(t *testing.T, cfg config.Config) {
	t.Helper()

	if len(cfg.Exclude) != 0 {
		t.Errorf("Exclude = %v, want empty", cfg.Exclude)
	}
	if cfg.IncludeHidden {
		t.Errorf("IncludeHidden = true, want false")
	}
	if cfg.History.MaxVisits != 1000 {
		t.Errorf("History.MaxVisits = %d, want 1000", cfg.History.MaxVisits)
	}
	if cfg.Keys.Vim {
		t.Errorf("Keys.Vim = true, want false")
	}
	if cfg.Picker.Layout != "list" {
		t.Errorf("Picker.Layout = %q, want %q", cfg.Picker.Layout, "list")
	}
}

func TestLoadFromMissingFileIsDefaults(t *testing.T) {
	cfg, err := config.LoadFrom(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("LoadFrom missing file: %v", err)
	}
	assertDefaults(t, cfg)
}

func TestLoadFromEmptyFileIsDefaults(t *testing.T) {
	cfg, err := config.LoadFrom(writeConfig(t, ""))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	assertDefaults(t, cfg)
}

func TestLoadFromEachKey(t *testing.T) {
	body := `exclude = ["archive", "/srv/scratch", "node_*"]
include_hidden = true
[history]
max_visits = 42
[keys]
vim = true
[picker]
layout = "list"
`
	cfg, err := config.LoadFrom(writeConfig(t, body))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if want := []string{"archive", "/srv/scratch", "node_*"}; !slices.Equal(cfg.Exclude, want) {
		t.Errorf("Exclude = %v, want %v", cfg.Exclude, want)
	}
	if !cfg.IncludeHidden {
		t.Errorf("IncludeHidden = false, want true")
	}
	if cfg.History.MaxVisits != 42 {
		t.Errorf("History.MaxVisits = %d, want 42", cfg.History.MaxVisits)
	}
	if !cfg.Keys.Vim {
		t.Errorf("Keys.Vim = false, want true")
	}
	if cfg.Picker.Layout != "list" {
		t.Errorf("Picker.Layout = %q, want %q", cfg.Picker.Layout, "list")
	}
}

func TestLoadFromExcludeTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	cfg, err := config.LoadFrom(writeConfig(t, `exclude = ["~/go/pkg/*", "~", "$HOME/x", "node_modules"]`+"\n"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	want := []string{filepath.Join(home, "go", "pkg", "*"), home, "$HOME/x", "node_modules"}
	if !slices.Equal(cfg.Exclude, want) {
		t.Errorf("Exclude = %v, want %v", cfg.Exclude, want)
	}
}

func TestLoadFromInvalidExcludePatternRejected(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, `exclude = ["["]`+"\n"))
	if err == nil {
		t.Fatal("LoadFrom invalid exclude pattern: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "exclude") {
		t.Errorf("error = %q, want it to name exclude", err.Error())
	}
}

func TestLoadFromUnknownKeyRejected(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, "bogus = true\n"))
	if err == nil {
		t.Fatal("LoadFrom unknown key: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error = %q, want it to name the key %q", err.Error(), "bogus")
	}
}

func TestLoadFromRemovedRootExplained(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, `root = "~/Developer"`+"\n"))
	if err == nil {
		t.Fatal("LoadFrom root: got nil error, want error")
	}
	for _, want := range []string{"root", "removed", "cdd scan", "line 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestLoadFromMaxVisitsZeroRejected(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, "[history]\nmax_visits = 0\n"))
	if err == nil {
		t.Fatal("LoadFrom max_visits = 0: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "max_visits") {
		t.Errorf("error = %q, want it to name max_visits", err.Error())
	}
}

func TestLoadFromUnknownPickerLayoutRejected(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, "[picker]\nlayout = \"fancy\"\n"))
	if err == nil {
		t.Fatal(`LoadFrom layout = "fancy": got nil error, want error`)
	}
	for _, want := range []string{"picker.layout", "fancy", `"list"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestLoadFromGroupedLayoutExplained(t *testing.T) {
	_, err := config.LoadFrom(writeConfig(t, "[picker]\nlayout = \"grouped\"\n"))
	if err == nil {
		t.Fatal(`LoadFrom layout = "grouped": got nil error, want error`)
	}
	for _, want := range []string{"picker.layout", "grouped", "removed", `"list"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}
