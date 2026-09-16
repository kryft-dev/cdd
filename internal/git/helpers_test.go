package git_test

import (
	"os"
	"os/exec"
	"testing"
)

// run executes a git command in dir with a test identity and no system or
// global config, failing the test on error.
func run(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME=/nonexistent-cdd-test-home",
		"GIT_AUTHOR_NAME=cdd test",
		"GIT_AUTHOR_EMAIL=cdd-test@example.com",
		"GIT_COMMITTER_NAME=cdd test",
		"GIT_COMMITTER_EMAIL=cdd-test@example.com",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newRepo creates an initialised git repository with a test identity
// configured locally, and returns its path.
func newRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	run(t, dir, "config", "user.name", "cdd test")
	run(t, dir, "config", "user.email", "cdd-test@example.com")
	return dir
}

// commit stages everything in dir and commits it.
func commit(t *testing.T, dir, message string) {
	t.Helper()

	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", message)
}

// writeFile writes content to a file at path relative to dir, creating it.
func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()

	full := dir + "/" + path
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}
