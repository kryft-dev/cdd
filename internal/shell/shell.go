// Package shell renders the Wrapper: the shell function that "cdd init"
// prints for the user to install into their fish, bash or zsh
// configuration. The Wrapper forwards a fixed passthrough list of
// subcommands straight to the cdd binary and, for anything else, captures
// the binary's Picker output and turns a non-empty result into a Jump.
package shell

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/cdd.fish templates/cdd.bash templates/cdd.zsh
var templatesFS embed.FS

// supported maps a shell name to the embedded template that renders its
// Wrapper.
var supported = map[string]string{
	"fish": "templates/cdd.fish",
	"bash": "templates/cdd.bash",
	"zsh":  "templates/cdd.zsh",
}

// data is the value passed to the Wrapper templates.
type data struct {
	// Passthrough lists the subcommand names the Wrapper forwards to the
	// cdd binary unchanged, alongside --help/-h/--version/-V.
	Passthrough []string
}

// Script renders the Wrapper for the given shell, one of "fish", "bash" or
// "zsh". passthrough is the list of subcommand names the Wrapper forwards
// to the cdd binary unchanged; the caller derives it from the cobra command
// tree. An unrecognized shell returns an error listing the three supported
// shells.
func Script(shell string, passthrough []string) (string, error) {
	path, ok := supported[shell]
	if !ok {
		return "", fmt.Errorf("unsupported shell %q: cdd init supports fish, bash, zsh", shell)
	}

	tmpl, err := template.ParseFS(templatesFS, path)
	if err != nil {
		return "", fmt.Errorf("parse %s Wrapper template: %w", shell, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data{Passthrough: passthrough}); err != nil {
		return "", fmt.Errorf("render %s Wrapper: %w", shell, err)
	}
	return buf.String(), nil
}
