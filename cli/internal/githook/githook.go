// Package githook installs and removes CommitIn's commit-msg git hook.
package githook

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileName is the git hook CommitIn manages.
const FileName = "commit-msg"

// Marker identifies a hook file as CommitIn's own, so init/uninstall never
// touch a hook they didn't write.
const Marker = "Installed by CommitIn (cmtin init). Do not edit by hand."

// Script is the hook CommitIn installs. It delegates to the cmtin binary; if
// cmtin isn't on PATH the hook skips itself (exit 0). Otherwise cmtin's own
// exit code decides the commit: `cmtin hook commit-msg` exits 0 to let a
// commit through (a good message, or any infra failure -- it fails open
// internally) and non-zero only to reject a genuinely bad message, which
// this script must NOT swallow.
const Script = `#!/bin/sh
# ` + Marker + `
# Regenerate with ` + "`cmtin init`" + `; remove with ` + "`cmtin uninstall`" + `.
if ! command -v cmtin >/dev/null 2>&1; then
  exit 0
fi
cmtin hook commit-msg "$1"
`

// GitDir returns the absolute path to the current repository's .git
// directory (resolving worktrees, GIT_DIR, etc. via git itself). It errors
// if the current directory isn't inside a git repository, or git isn't
// installed.
func GitDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or any parent directory)")
	}
	return strings.TrimSpace(string(out)), nil
}

// Path returns the commit-msg hook's path under gitDir.
func Path(gitDir string) string {
	return filepath.Join(gitDir, "hooks", FileName)
}

// Installed reports whether CommitIn's hook is currently at path.
func Installed(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bytes.Contains(b, []byte(Marker)), nil
}

// Install writes the hook script into gitDir's hooks directory. It refuses
// to overwrite a commit-msg hook that isn't CommitIn's own.
func Install(gitDir string) error {
	path := Path(gitDir)

	ours, err := Installed(path)
	if err != nil {
		return fmt.Errorf("check existing hook: %w", err)
	}
	if !ours {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists and wasn't installed by CommitIn -- back it up or remove it, then run `cmtin init` again", path)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(Script), 0o755); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}
	// WriteFile only applies the mode when creating a new file; force it on
	// every install so a stricter umask can't leave the hook non-executable.
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("make hook executable: %w", err)
	}
	return nil
}

// Uninstall removes CommitIn's hook from gitDir's hooks directory, leaving
// any non-CommitIn hook alone. Reports whether a hook was actually removed.
func Uninstall(gitDir string) (bool, error) {
	path := Path(gitDir)

	ours, err := Installed(path)
	if err != nil {
		return false, fmt.Errorf("check existing hook: %w", err)
	}
	if !ours {
		if _, err := os.Stat(path); err == nil {
			return false, fmt.Errorf("%s exists but wasn't installed by CommitIn -- not removing it", path)
		}
		return false, nil
	}

	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("remove hook: %w", err)
	}
	return true, nil
}
