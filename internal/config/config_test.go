package config_test

import (
	"os"
	"path/filepath"
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

func TestLoadFromMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("LoadFrom missing file: got nil error, want error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error = %q, want it to name the path %q", err.Error(), path)
	}
	if !strings.Contains(err.Error(), `root = "~/Developer"`) {
		t.Errorf("error = %q, want it to include the example config", err.Error())
	}
}

func TestLoadFromMinimalAppliesDefaults(t *testing.T) {
	root := t.TempDir()
	path := writeConfig(t, `root = "`+root+`"`+"\n")

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Root != root {
		t.Errorf("Root = %q, want %q", cfg.Root, root)
	}
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
}

func TestLoadFromEachKey(t *testing.T) {
	root := t.TempDir()
	body := `root = "` + root + `"
exclude = ["archive", "tools/scratch", "*/node_modules"]
include_hidden = true
[history]
max_visits = 42
[keys]
vim = true
`
	path := writeConfig(t, body)

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Root != root {
		t.Errorf("Root = %q, want %q", cfg.Root, root)
	}
	wantExclude := []string{"archive", "tools/scratch", "*/node_modules"}
	if len(cfg.Exclude) != len(wantExclude) {
		t.Fatalf("Exclude = %v, want %v", cfg.Exclude, wantExclude)
	}
	for i, want := range wantExclude {
		if cfg.Exclude[i] != want {
			t.Errorf("Exclude[%d] = %q, want %q", i, cfg.Exclude[i], want)
		}
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
}

func TestLoadFromTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	// Expand against "~" itself (the home directory), which always
	// exists, rather than fabricating a subdirectory under the real home.
	path := writeConfig(t, `root = "~"`+"\n")

	cfg, err2 := config.LoadFrom(path)
	if err2 != nil {
		t.Fatalf("LoadFrom: %v", err2)
	}
	if cfg.Root != home {
		t.Errorf("Root = %q, want %q", cfg.Root, home)
	}
}

func TestLoadFromUnknownKeyRejected(t *testing.T) {
	root := t.TempDir()
	body := `root = "` + root + `"
bogus = true
`
	path := writeConfig(t, body)

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("LoadFrom unknown key: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error = %q, want it to name the key %q", err.Error(), "bogus")
	}
}

func TestLoadFromMaxVisitsZeroRejected(t *testing.T) {
	root := t.TempDir()
	body := `root = "` + root + `"
[history]
max_visits = 0
`
	path := writeConfig(t, body)

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("LoadFrom max_visits = 0: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "max_visits") {
		t.Errorf("error = %q, want it to name max_visits", err.Error())
	}
}

func TestLoadFromRootNotDirectoryRejected(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "root-*")
	if err != nil {
		t.Fatalf("os.CreateTemp: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	body := `root = "` + file.Name() + `"` + "\n"
	path := writeConfig(t, body)

	_, err = config.LoadFrom(path)
	if err == nil {
		t.Fatal("LoadFrom root not a directory: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error = %q, want it to say root is not a directory", err.Error())
	}
}

func TestLoadFromRootMissing(t *testing.T) {
	path := writeConfig(t, `include_hidden = true`+"\n")

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("LoadFrom missing root: got nil error, want error")
	}
	if !strings.Contains(err.Error(), "root") {
		t.Errorf("error = %q, want it to mention root", err.Error())
	}
	if !strings.Contains(err.Error(), `root = "~/Developer"`) {
		t.Errorf("error = %q, want it to include the example config", err.Error())
	}
}

func TestExampleConfig(t *testing.T) {
	got := config.ExampleConfig()
	for _, want := range []string{"root =", "exclude =", "include_hidden =", "[history]", "max_visits =", "[keys]", "vim ="} {
		if !strings.Contains(got, want) {
			t.Errorf("ExampleConfig() = %q, want it to contain %q", got, want)
		}
	}
}
