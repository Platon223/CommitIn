package gitcli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestDiffCached(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")

	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "file.txt")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	diff, err := DiffCached()
	if err != nil {
		t.Fatalf("DiffCached: %v", err)
	}
	if !strings.Contains(diff, "file.txt") || !strings.Contains(diff, "+hello") {
		t.Fatalf("diff missing expected content: %q", diff)
	}
}

func TestDiffCachedEmpty(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	diff, err := DiffCached()
	if err != nil {
		t.Fatalf("DiffCached: %v", err)
	}
	if diff != "" {
		t.Fatalf("expected empty diff, got %q", diff)
	}
}
