package shell_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/shell"
)

var passthrough = []string{"init", "scan", "version", "help", "completion"}

// TestScriptContainsWrapper checks that each supported shell's Wrapper
// contains the cdd function, the passthrough list, the non-empty guard, and
// the shell's cd form.
func TestScriptContainsWrapper(t *testing.T) {
	tests := []struct {
		name   string
		shell  string
		cdForm string
	}{
		{name: "fish", shell: "fish", cdForm: "cd -- $result"},
		{name: "bash", shell: "bash", cdForm: `\builtin cd -- "$result"`},
		{name: "zsh", shell: "zsh", cdForm: `\builtin cd -- "$result"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shell.Script(tt.shell, passthrough)
			if err != nil {
				t.Fatalf("Script(%q, ...) unexpected error: %v", tt.shell, err)
			}

			if !strings.Contains(got, "cdd") {
				t.Errorf("Script(%q, ...) missing the cdd function name", tt.shell)
			}
			for _, cmd := range passthrough {
				if !strings.Contains(got, cmd) {
					t.Errorf("Script(%q, ...) missing passthrough command %q", tt.shell, cmd)
				}
			}
			if !strings.Contains(got, "-n") && !strings.Contains(got, "test -n") {
				// bash/zsh use "[ -n", fish uses "test -n"
				if !strings.Contains(got, `[ -n "$result" ]`) {
					t.Errorf("Script(%q, ...) missing the non-empty guard", tt.shell)
				}
			}
			if !strings.Contains(got, tt.cdForm) {
				t.Errorf("Script(%q, ...) missing the cd form %q", tt.shell, tt.cdForm)
			}
		})
	}
}

// TestScriptUnknownShell checks that an unrecognized shell name returns an
// error listing the three supported shells.
func TestScriptUnknownShell(t *testing.T) {
	_, err := shell.Script("powershell", passthrough)
	if err == nil {
		t.Fatal("Script(\"powershell\", ...) expected an error, got nil")
	}
	for _, want := range []string{"fish", "bash", "zsh"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Script error = %q, want it to mention %q", err.Error(), want)
		}
	}
}

// TestScriptExecution installs a fake cdd binary on PATH that prints a
// path (or nothing) to stdout, sources the Wrapper for each installed
// shell, calls the cdd function, and asserts the working directory changes
// on a non-empty result and stays put on an empty one.
func TestScriptExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wrapper execution tests target POSIX shells")
	}

	target := t.TempDir()
	nonEmptyBin := buildFakeCDD(t, target)
	emptyBin := buildFakeCDD(t, "")

	tests := []struct {
		name    string
		shell   string
		runFile func(t *testing.T, dir, script, fakeBinDir string) (stdout string, err error)
	}{
		{name: "fish", shell: "fish", runFile: runFish},
		{name: "bash", shell: "bash", runFile: runBash},
		{name: "zsh", shell: "zsh", runFile: runZsh},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/jumps on a non-empty result", func(t *testing.T) {
			startDir := t.TempDir()
			script, err := shell.Script(tt.shell, passthrough)
			if err != nil {
				t.Fatalf("Script(%q, ...): %v", tt.shell, err)
			}

			out, err := tt.runFile(t, startDir, script, filepath.Dir(nonEmptyBin))
			if err != nil {
				t.Fatalf("running %s Wrapper: %v (output: %q)", tt.shell, err, out)
			}
			if strings.TrimSpace(out) != target {
				t.Errorf("%s Wrapper pwd = %q, want %q", tt.shell, strings.TrimSpace(out), target)
			}
		})

		t.Run(tt.name+"/stays put on an empty result", func(t *testing.T) {
			startDir := t.TempDir()
			script, err := shell.Script(tt.shell, passthrough)
			if err != nil {
				t.Fatalf("Script(%q, ...): %v", tt.shell, err)
			}

			out, err := tt.runFile(t, startDir, script, filepath.Dir(emptyBin))
			if err != nil {
				t.Fatalf("running %s Wrapper: %v (output: %q)", tt.shell, err, out)
			}
			// Resolve symlinks (e.g. macOS /tmp -> /private/tmp) so the
			// comparison holds regardless of how the shell reports pwd.
			wantDir, err := filepath.EvalSymlinks(startDir)
			if err != nil {
				t.Fatalf("resolve startDir: %v", err)
			}
			gotDir, err := filepath.EvalSymlinks(strings.TrimSpace(out))
			if err != nil {
				t.Fatalf("resolve pwd output %q: %v", strings.TrimSpace(out), err)
			}
			if gotDir != wantDir {
				t.Errorf("%s Wrapper pwd = %q, want %q (unchanged)", tt.shell, gotDir, wantDir)
			}
		})
	}
}

// buildFakeCDD builds a fake cdd binary that, when invoked as
// "cdd pick ...", prints the given result (or nothing, when result is
// empty). It is used in place of the real binary to test the Wrapper's
// capture-guard-cd behaviour in isolation.
func buildFakeCDD(t *testing.T, result string) string {
	t.Helper()

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "main.go")
	source := `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "pick" {
		fmt.Println(` + "`" + result + "`" + `)
		return
	}
}
`
	if err := os.WriteFile(src, []byte(source), 0o644); err != nil {
		t.Fatalf("write fake cdd source: %v", err)
	}

	binDir := t.TempDir()
	bin := filepath.Join(binDir, "cdd")
	cmd := exec.Command("go", "build", "-o", bin, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake cdd: %v (output: %q)", err, out)
	}
	return bin
}

// runFish sources the Wrapper script, calls "cdd" with the fake binary on
// PATH, and prints the resulting working directory.
func runFish(t *testing.T, dir, script, fakeBinDir string) (string, error) {
	t.Helper()
	return runShell(t, "fish", dir, fakeBinDir, script+"\ncdd\npwd\n")
}

// runBash sources the Wrapper script, calls "cdd" with the fake binary on
// PATH, and prints the resulting working directory.
func runBash(t *testing.T, dir, script, fakeBinDir string) (string, error) {
	t.Helper()
	return runShell(t, "bash", dir, fakeBinDir, script+"\ncdd\npwd\n")
}

// runZsh sources the Wrapper script, calls "cdd" with the fake binary on
// PATH, and prints the resulting working directory.
func runZsh(t *testing.T, dir, script, fakeBinDir string) (string, error) {
	t.Helper()
	return runShell(t, "zsh", dir, fakeBinDir, script+"\ncdd\npwd\n")
}

// runShell writes stdin to the named shell binary with fakeBinDir prepended
// to PATH and dir as the starting working directory, and returns stdout.
func runShell(t *testing.T, shellName, dir, fakeBinDir, stdin string) (string, error) {
	t.Helper()

	shellPath, err := exec.LookPath(shellName)
	if err != nil {
		return "", errors.New(shellName + " not installed: " + err.Error())
	}

	cmd := exec.Command(shellPath)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), "PATH="+fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := cmd.CombinedOutput()
	return string(out), err
}
