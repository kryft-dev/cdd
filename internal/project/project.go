// Package project discovers Projects under a Root: every directory directly
// under Root is a Kind, and every directory directly under a Kind is a
// Project.
package project

import (
	"path"
	"path/filepath"
)

// Project is a directory exactly one level below a Kind, identified by its
// path relative to Root.
type Project struct {
	Kind string
	Name string
}

// Rel returns the Project's path relative to Root, as "kind/name".
func (p Project) Rel() string {
	return path.Join(p.Kind, p.Name)
}

// Abs returns the Project's absolute path given root, the Root it was
// discovered under.
func (p Project) Abs(root string) string {
	return filepath.Join(root, p.Kind, p.Name)
}
