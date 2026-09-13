// Package gitcli runs the handful of git subcommands the CLI depends on.
package gitcli

import (
	"bytes"
	"fmt"
	"os/exec"
)

// DiffCached returns the staged diff (`git diff --cached`) for the
// repository rooted at the current working directory.
func DiffCached() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff --cached: %w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
