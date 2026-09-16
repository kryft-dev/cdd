// Command cdd is the entry point for the cdd terminal UI.
package main

import (
	"os"

	"github.com/kryft-dev/cdd/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
