package cdd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd"
	"github.com/kryft-dev/cdd/internal/config"
)

// TestConfigExampleLoads checks the embedded example is a valid config, as
// shipped and with every commented example uncommented.
func TestConfigExampleLoads(t *testing.T) {
	t.Run("as shipped", func(t *testing.T) {
		assertLoads(t, string(cdd.ConfigExample))
	})

	// The Action examples are commented out, so uncomment every one: they
	// must also load, all together.
	t.Run("actions uncommented", func(t *testing.T) {
		_, actions, found := strings.Cut(string(cdd.ConfigExample), "# [actions.code]")
		if !found {
			t.Fatal("no [actions.code] example found")
		}
		var b strings.Builder
		b.WriteString("[actions.code]\n")
		for _, line := range strings.Split(actions, "\n") {
			if rest, ok := strings.CutPrefix(line, "# "); ok && strings.ContainsAny(rest, "=[") {
				b.WriteString(rest + "\n")
			}
		}
		assertLoads(t, b.String())
	})
}

func assertLoads(t *testing.T, body string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := config.LoadFrom(path); err != nil {
		t.Fatalf("LoadFrom: %v\n%s", err, body)
	}
}
