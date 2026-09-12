// Command cmtin is the CommitIn CLI.
package main

import (
	"os"

	"github.com/Platon223/commitin/cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
