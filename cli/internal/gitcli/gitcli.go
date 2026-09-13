// Package gitcli runs the handful of git subcommands the CLI depends on.
package gitcli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// DefaultMaxDiffLines is the line cap TruncateDiff uses for the commit-msg
// hook's own output. (Separate from whatever cap Week 3's Claude prompt
// ends up using -- that's a token/cost budget, this is terminal usability.)
const DefaultMaxDiffLines = 400

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

// TruncateDiff caps diff to at most maxLines lines, reporting whether it had
// to cut anything.
func TruncateDiff(diff string, maxLines int) (result string, truncated bool) {
	if diff == "" {
		return diff, false
	}
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff, false
	}
	return strings.Join(lines[:maxLines], "\n") + "\n", true
}

// IsMerging reports whether the repository is currently completing a merge
// (MERGE_HEAD exists). A merge commit's `diff --cached` compares the index
// against only one of its two parents, so it doesn't represent "this
// commit's changes" the way a normal commit's diff does -- callers should
// skip analysis rather than show a possibly huge, misleading diff.
func IsMerging() bool {
	return exec.Command("git", "rev-parse", "--verify", "-q", "MERGE_HEAD").Run() == nil
}
