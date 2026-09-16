package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/project"
)

// mkDirs creates each rel path as a directory tree under root.
func mkDirs(t *testing.T, root string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", rel, err)
		}
	}
}

func TestDiscoverNestedLayout(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, "tools/cdd", "tools/scratch", "archive/old")

	got, err := project.Discover(root, nil, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	want := []project.Project{
		{Kind: "archive", Name: "old"},
		{Kind: "tools", Name: "cdd"},
		{Kind: "tools", Name: "scratch"},
	}
	assertProjects(t, got, want)
}

func TestDiscoverHiddenSkippedByDefault(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, ".hidden/proj", "tools/.hidden", "tools/visible")

	got, err := project.Discover(root, nil, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	want := []project.Project{
		{Kind: "tools", Name: "visible"},
	}
	assertProjects(t, got, want)
}

func TestDiscoverIncludeHidden(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, ".hidden/proj", "tools/.hidden", "tools/visible")

	got, err := project.Discover(root, nil, true)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	want := []project.Project{
		{Kind: ".hidden", Name: "proj"},
		{Kind: "tools", Name: ".hidden"},
		{Kind: "tools", Name: "visible"},
	}
	assertProjects(t, got, want)
}

func TestDiscoverExcludePatterns(t *testing.T) {
	tests := []struct {
		name    string
		exclude []string
		want    []project.Project
	}{
		{
			name:    "kind name excludes whole kind",
			exclude: []string{"archive"},
			want: []project.Project{
				{Kind: "tools", Name: "cdd"},
				{Kind: "tools", Name: "scratch"},
			},
		},
		{
			name:    "kind/name excludes one project",
			exclude: []string{"tools/scratch"},
			want: []project.Project{
				{Kind: "archive", Name: "old"},
				{Kind: "tools", Name: "cdd"},
			},
		},
		{
			name:    "kind/* excludes all projects under a kind",
			exclude: []string{"archive/*"},
			want: []project.Project{
				{Kind: "tools", Name: "cdd"},
				{Kind: "tools", Name: "scratch"},
			},
		},
		{
			name:    "*/name excludes a project name under any kind",
			exclude: []string{"*/scratch"},
			want: []project.Project{
				{Kind: "archive", Name: "old"},
				{Kind: "tools", Name: "cdd"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			mkDirs(t, root, "tools/cdd", "tools/scratch", "archive/old")

			got, err := project.Discover(root, tt.exclude, false)
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			assertProjects(t, got, tt.want)
		})
	}
}

func TestDiscoverInvalidPattern(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, "tools/cdd")

	_, err := project.Discover(root, []string{"["}, false)
	if err == nil {
		t.Fatal("Discover with invalid pattern: want error, got nil")
	}
}

func TestDiscoverIgnoresNonDirectories(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, "tools")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	mkDirs(t, root, "tools/cdd")

	got, err := project.Discover(root, nil, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []project.Project{{Kind: "tools", Name: "cdd"}}
	assertProjects(t, got, want)
}

func TestDiscoverSymlinkedDirectoryCountsAsDirectory(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	mkDirs(t, elsewhere, "real-kind/real-project")

	if err := os.Symlink(filepath.Join(elsewhere, "real-kind"), filepath.Join(root, "linked-kind")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	got, err := project.Discover(root, nil, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []project.Project{{Kind: "linked-kind", Name: "real-project"}}
	assertProjects(t, got, want)
}

func TestDiscoverEmptyKindYieldsNothing(t *testing.T) {
	root := t.TempDir()
	mkDirs(t, root, "empty-kind")

	got, err := project.Discover(root, nil, false)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	assertProjects(t, got, nil)
}

// assertProjects compares got against want by Rel(), since ordering and
// content are what Discover promises.
func assertProjects(t *testing.T, got, want []project.Project) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Discover() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Discover()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
