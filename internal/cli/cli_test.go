package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/cli"
)

// runCLI drives cli.Run in-process with args and returns the exit code
// plus captured stdout and stderr.
func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()

	var out, err bytes.Buffer
	code = cli.Run(args, &out, &err)
	return code, out.String(), err.String()
}

// TestVersion checks "cdd version" and the "cdd --version" / "cdd -V"
// aliases all print the same dev-build version line and exit 0.
func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "version subcommand", args: []string{"version"}},
		{name: "long flag", args: []string{"--version"}},
		{name: "short flag", args: []string{"-V"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tt.args...)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr)
			}
			want := "cdd dev (dev)\n"
			if stdout != want {
				t.Errorf("stdout = %q, want %q", stdout, want)
			}
		})
	}
}

// TestUnknownCommand checks that an unrecognized subcommand exits 2 with a
// usage message on stderr and nothing on stdout.
func TestUnknownCommand(t *testing.T) {
	code, stdout, stderr := runCLI(t, "bogus")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr = %q, want it to contain %q", stderr, "unknown command")
	}
}

// TestUnknownFlag checks that an unrecognized flag exits 2.
func TestUnknownFlag(t *testing.T) {
	code, _, stderr := runCLI(t, "--bogus")

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
}

// TestInitUsageErrors checks "cdd init" exits 2, listing the supported
// shells, when given no argument, more than one argument, or an
// unsupported shell.
func TestInitUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no shell", args: []string{"init"}},
		{name: "two shells", args: []string{"init", "fish", "bash"}},
		{name: "unsupported shell", args: []string{"init", "powershell"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tt.args...)

			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			for _, shell := range []string{"fish", "bash", "zsh"} {
				if !strings.Contains(stderr, shell) {
					t.Errorf("stderr = %q, want it to list %q", stderr, shell)
				}
			}
		})
	}
}

// TestInitPassthrough checks that "cdd init fish" prints a script naming
// every top-level command except "pick" in its pass-through list, derived
// from the cobra command tree.
func TestInitPassthrough(t *testing.T) {
	code, stdout, stderr := runCLI(t, "init", "fish")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}

	for _, name := range []string{"init", "scan", "version", "help", "completion"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("script does not mention pass-through command %q:\n%s", name, stdout)
		}
	}

	// "pick" itself is the Wrapper's Jump path, not a pass-through name;
	// it must not appear in the case pattern the Wrapper switches on.
	if strings.Contains(stdout, "case pick") {
		t.Errorf("script lists \"pick\" as a pass-through command:\n%s", stdout)
	}
}

// TestInitEachShell checks that "cdd init" succeeds for each supported
// shell and produces non-empty output.
func TestInitEachShell(t *testing.T) {
	for _, shell := range []string{"fish", "bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, "init", shell)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
			}
			if stdout == "" {
				t.Error("stdout is empty, want a rendered Wrapper script")
			}
		})
	}
}

// TestPickUsageError checks that "cdd pick" with more than one argument
// exits 2.
func TestPickUsageError(t *testing.T) {
	code, _, _ := runCLI(t, "pick", "one", "two")
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

// TestScanUsageError checks that "cdd scan" with an argument exits 2.
func TestScanUsageError(t *testing.T) {
	code, _, _ := runCLI(t, "scan", "extra")
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

// TestHelp checks that "cdd help" and "cdd --help" both exit 0 and print
// something to stdout.
func TestHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "help command", args: []string{"help"}},
		{name: "long flag", args: []string{"--help"}},
		{name: "short flag", args: []string{"-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tt.args...)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr)
			}
			if stdout == "" {
				t.Error("stdout is empty, want help text")
			}
		})
	}
}
