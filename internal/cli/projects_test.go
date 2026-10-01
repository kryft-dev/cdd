package cli_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kryft-dev/cdd/internal/history"
	"github.com/kryft-dev/cdd/internal/project"
)

// isolate points cdd's data and config at fresh temp directories and
// returns a directory to use as a Project.
func isolate(t *testing.T) string {
	t.Helper()

	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return dir
}

// marks reads the projects file under the isolated XDG_DATA_HOME.
func marks(t *testing.T) project.Marks {
	t.Helper()

	path, err := project.StorePath()
	if err != nil {
		t.Fatalf("StorePath: %v", err)
	}
	m, err := project.OpenStore(path).Marks()
	if err != nil {
		t.Fatalf("Marks: %v", err)
	}
	return m
}

// visited returns the Projects in the isolated History.
func visited(t *testing.T) []string {
	t.Helper()

	path, err := history.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	h, err := history.Open(path, 1000)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	latest, err := h.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	var out []string
	for _, v := range latest {
		out = append(out, v.Project)
	}
	return out
}

func TestAdd_RecordsProjectAndVisit(t *testing.T) {
	dir := isolate(t)

	code, stdout, stderr := runCLI(t, "add", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if want := "Added " + dir + "\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if !marks(t).IsAdded(dir) {
		t.Error("the directory should be Added")
	}
	if got := visited(t); !slices.Equal(got, []string{dir}) {
		t.Errorf("History = %v, want [%s]", got, dir)
	}
}

func TestAdd_DefaultsToCurrentDirectory(t *testing.T) {
	dir := isolate(t)
	t.Chdir(dir)
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	if code, _, stderr := runCLI(t, "add"); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !marks(t).IsAdded(want) {
		t.Errorf("the current directory %s should be Added", want)
	}
}

func TestAdd_ResolvesRelativeAndCleansPath(t *testing.T) {
	dir := isolate(t)
	t.Chdir(filepath.Dir(dir))
	want, err := filepath.Abs("notes")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	if code, _, stderr := runCLI(t, "add", "./notes/../notes/"); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !marks(t).IsAdded(want) {
		t.Errorf("%s should be Added", want)
	}
}

func TestAdd_MissingDirectoryFails(t *testing.T) {
	dir := isolate(t)
	missing := filepath.Join(dir, "nope")

	code, _, stderr := runCLI(t, "add", missing)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "not a directory") {
		t.Errorf("stderr = %q, want it to say %q", stderr, "not a directory")
	}
	if marks(t).IsAdded(missing) {
		t.Error("a missing directory must not be Added")
	}
}

func TestAdd_FileFails(t *testing.T) {
	dir := isolate(t)
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if code, _, _ := runCLI(t, "add", file); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestAdd_TooManyArgsIsUsageError(t *testing.T) {
	isolate(t)

	if code, _, _ := runCLI(t, "add", "a", "b"); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestForget_ThenAddBringsItBack(t *testing.T) {
	dir := isolate(t)

	code, stdout, stderr := runCLI(t, "forget", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if want := "Forgot " + dir + "\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if !marks(t).IsForgotten(dir) {
		t.Error("the directory should be Forgotten")
	}

	if code, _, stderr := runCLI(t, "add", dir); code != 0 {
		t.Fatalf("add: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if m := marks(t); m.IsForgotten(dir) || !m.IsAdded(dir) {
		t.Error("add should bring a Forgotten Project back")
	}
}

func TestForget_AcceptsAMissingDirectory(t *testing.T) {
	dir := isolate(t)
	missing := filepath.Join(dir, "gone")

	if code, _, stderr := runCLI(t, "forget", missing); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !marks(t).IsForgotten(missing) {
		t.Error("a Project whose directory is gone can still be Forgotten")
	}
}

func TestForget_DefaultsToCurrentDirectory(t *testing.T) {
	dir := isolate(t)
	t.Chdir(dir)
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	if code, _, stderr := runCLI(t, "forget"); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !marks(t).IsForgotten(want) {
		t.Errorf("the current directory %s should be Forgotten", want)
	}
}

func TestInitPassthrough_IncludesAddAndForget(t *testing.T) {
	_, stdout, _ := runCLI(t, "init", "bash")

	for _, name := range []string{"add|", "forget|"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("bash Wrapper does not pass %q through:\n%s", name, stdout)
		}
	}
}
