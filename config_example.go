// Package cdd holds config.example.toml, the commented config the release
// ships and `cdd config init` writes. The file lives at the repository
// root, where a release tarball picks it up, and //go:embed cannot reach
// above a package's own directory, so this root package embeds it.
package cdd

import _ "embed"

// ConfigExample is config.example.toml.
//
//go:embed config.example.toml
var ConfigExample []byte
