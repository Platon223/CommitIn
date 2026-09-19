package gitcli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// chdirTemp creates a fresh git repo, chdirs into it for the duration of the
// test, and restores the original working directory on cleanup.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDiffCached(t *testing.T) {
	dir := chdirTemp(t)
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "file.txt")

	diff, err := DiffCached()
	if err != nil {
		t.Fatalf("DiffCached: %v", err)
	}
	if !strings.Contains(diff, "file.txt") || !strings.Contains(diff, "+hello") {
		t.Fatalf("diff missing expected content: %q", diff)
	}
}

func TestDiffCachedEmpty(t *testing.T) {
	chdirTemp(t)

	diff, err := DiffCached()
	if err != nil {
		t.Fatalf("DiffCached: %v", err)
	}
	if diff != "" {
		t.Fatalf("expected empty diff, got %q", diff)
	}
}

func TestRepoName(t *testing.T) {
	dir := chdirTemp(t)
	// git resolves symlinks in --show-toplevel; compare against the
	// resolved path so this doesn't flake on a symlinked temp dir.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	name, err := RepoName()
	if err != nil {
		t.Fatalf("RepoName: %v", err)
	}
	if want := filepath.Base(resolved); name != want {
		t.Fatalf("RepoName() = %q, want %q", name, want)
	}
}

func TestTruncateDiffUnderLimit(t *testing.T) {
	diff := "a\nb\nc\n"
	got, truncated := TruncateDiff(diff, 10)
	if truncated {
		t.Fatal("truncated a diff under the limit")
	}
	if got != diff {
		t.Fatalf("got %q, want unchanged %q", got, diff)
	}
}

func TestTruncateDiffOverLimit(t *testing.T) {
	diff := "1\n2\n3\n4\n5\n"
	got, truncated := TruncateDiff(diff, 3)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if got != "1\n2\n3\n" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncateDiffEmpty(t *testing.T) {
	got, truncated := TruncateDiff("", 10)
	if truncated || got != "" {
		t.Fatalf("got (%q, %v), want (\"\", false)", got, truncated)
	}
}

func TestIsMergingFalseNormally(t *testing.T) {
	chdirTemp(t)
	if IsMerging() {
		t.Fatal("IsMerging() = true in a fresh repo with no merge in progress")
	}
}

func TestIsMergingTrueWithMergeHead(t *testing.T) {
	dir := chdirTemp(t)
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f")
	runGit(t, dir, "commit", "-q", "-m", "init")
	sha := strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))

	if err := os.WriteFile(filepath.Join(dir, ".git", "MERGE_HEAD"), []byte(sha+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !IsMerging() {
		t.Fatal("IsMerging() = false with MERGE_HEAD present")
	}
}
