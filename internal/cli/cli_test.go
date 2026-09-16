package cli_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestVersionString builds the cdd binary and runs "cdd version" to check
// the unflagged (dev) version output, since versionString's package-level
// vars are only reachable through the built binary or by duplicating the
// build's ldflags in-process.
func TestVersionString(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "dev build prints dev version",
			want: "cdd dev (dev)\n",
		},
	}

	bin := buildCDD(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command(bin, "version").CombinedOutput()
			if err != nil {
				t.Fatalf("cdd version: %v (output: %q)", err, out)
			}
			if got := string(out); got != tt.want {
				t.Errorf("cdd version = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExecuteUnknownCommand checks that an unrecognized subcommand exits 2
// with a usage line on stderr.
func TestExecuteUnknownCommand(t *testing.T) {
	bin := buildCDD(t)

	cmd := exec.Command(bin, "bogus")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("cdd bogus: expected *exec.ExitError, got %v", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), "usage:")
	}
}

// buildCDD builds the cmd/cdd binary into a temp directory and returns its
// path.
func buildCDD(t *testing.T) string {
	t.Helper()

	bin := t.TempDir() + "/cdd"
	build := exec.Command("go", "build", "-o", bin, "../../cmd/cdd")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build cmd/cdd: %v (output: %q)", err, out)
	}
	return bin
}
