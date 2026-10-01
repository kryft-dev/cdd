package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd"
)

// isolateConfig points XDG_CONFIG_HOME at a fresh temp directory and
// returns the config path under it, which does not exist yet.
func isolateConfig(t *testing.T) string {
	t.Helper()

	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	return filepath.Join(xdg, "cdd", "config.toml")
}

// readFile returns path's contents.
func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

// TestConfigPath checks "cdd config path" prints the config path, whether
// or not the file exists.
func TestConfigPath(t *testing.T) {
	want := isolateConfig(t)

	code, stdout, stderr := runCLI(t, "config", "path")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := strings.TrimSpace(stdout); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestConfigInit checks "cdd config init" writes the example and creates
// the directories, refuses to replace an existing file, and replaces it
// with --force.
func TestConfigInit(t *testing.T) {
	path := isolateConfig(t)

	code, stdout, stderr := runCLI(t, "config", "init")
	if code != 0 {
		t.Fatalf("init: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stdout, path) {
		t.Errorf("init: stdout = %q, want it to name %q", stdout, path)
	}
	if got := readFile(t, path); got != string(cdd.ConfigExample) {
		t.Errorf("init wrote %q, want the example config", got)
	}

	if err := os.WriteFile(path, []byte("include_hidden = true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	code, _, stderr = runCLI(t, "config", "init")
	if code != 1 {
		t.Errorf("init over a file: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("init over a file: stderr = %q, want it to mention --force", stderr)
	}
	if got := readFile(t, path); got != "include_hidden = true\n" {
		t.Errorf("init over a file changed it to %q", got)
	}

	code, _, stderr = runCLI(t, "config", "init", "--force")
	if code != 0 {
		t.Fatalf("init --force: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := readFile(t, path); got != string(cdd.ConfigExample) {
		t.Errorf("init --force wrote %q, want the example config", got)
	}
}

// fakeEditor writes an editor script that appends its argument to a log
// beside it, and returns the script and log paths.
func fakeEditor(t *testing.T) (script, log string) {
	t.Helper()

	dir := t.TempDir()
	log = filepath.Join(dir, "log")
	script = filepath.Join(dir, "editor")
	body := "#!/bin/sh\necho \"$0 $*\" >> " + log + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return script, log
}

// TestConfigEdit checks "cdd config edit" runs $VISUAL, else $EDITOR, on
// the config path, writing the example first when the file is missing and
// leaving an existing file as it is.
func TestConfigEdit(t *testing.T) {
	path := isolateConfig(t)
	script, log := fakeEditor(t)

	t.Setenv("VISUAL", script+" -w")
	t.Setenv("EDITOR", "false")

	if code, _, stderr := runCLI(t, "config", "edit"); code != 0 {
		t.Fatalf("edit: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := readFile(t, path); got != string(cdd.ConfigExample) {
		t.Errorf("edit on a missing file wrote %q, want the example config", got)
	}
	if got, want := readFile(t, log), script+" -w "+path+"\n"; got != want {
		t.Errorf("$VISUAL ran %q, want %q", got, want)
	}

	if err := os.WriteFile(path, []byte("include_hidden = true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", script)
	if code, _, stderr := runCLI(t, "config", "edit"); code != 0 {
		t.Fatalf("edit: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := readFile(t, path); got != "include_hidden = true\n" {
		t.Errorf("edit on an existing file changed it to %q", got)
	}
	if got := readFile(t, log); !strings.HasSuffix(got, script+" "+path+"\n") {
		t.Errorf("$EDITOR did not run on %s: log %q", path, got)
	}
}

// TestConfigEditFailure checks a failing editor makes "cdd config edit"
// exit 1.
func TestConfigEditFailure(t *testing.T) {
	isolateConfig(t)
	t.Setenv("VISUAL", "false")

	if code, _, _ := runCLI(t, "config", "edit"); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
