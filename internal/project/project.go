// Package project finds Projects, directories holding a git repository, by
// walking the directories a Scan is given, and tells whether a path still
// holds one.
package project

import (
	"os"
	"path/filepath"
)

// gitEntry is the name whose presence, as a directory (a clone) or a file
// (a worktree or submodule), makes a directory a Project.
const gitEntry = ".git"

// IsRepo reports whether dir holds a git repository: a .git directory or
// file directly inside it.
func IsRepo(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, gitEntry))
	return err == nil
}
