package project_test

import (
	"path/filepath"
	"testing"

	"github.com/kryft-dev/cdd/internal/project"
)

func TestProjectRel(t *testing.T) {
	p := project.Project{Kind: "tools", Name: "cdd"}
	if got, want := p.Rel(), "tools/cdd"; got != want {
		t.Errorf("Rel() = %q, want %q", got, want)
	}
}

func TestProjectAbs(t *testing.T) {
	p := project.Project{Kind: "tools", Name: "cdd"}
	root := "/home/user/Developer"
	if got, want := p.Abs(root), filepath.Join(root, "tools", "cdd"); got != want {
		t.Errorf("Abs(%q) = %q, want %q", root, got, want)
	}
}
