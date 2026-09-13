package githook

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallFreshAndIdempotent(t *testing.T) {
	gitDir := t.TempDir()

	if err := Install(gitDir); err != nil {
		t.Fatalf("Install: %v", err)
	}
	path := Path(gitDir)

	ours, err := Installed(path)
	if err != nil {
		t.Fatalf("Installed: %v", err)
	}
	if !ours {
		t.Fatal("Installed() = false right after Install")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("hook is not owner-executable: mode %o", info.Mode().Perm())
		}
	}

	// Re-running init should just refresh our own hook, not error.
	if err := Install(gitDir); err != nil {
		t.Fatalf("second Install (idempotent re-init): %v", err)
	}
}

func TestInstallRefusesForeignHook(t *testing.T) {
	gitDir := t.TempDir()
	path := Path(gitDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const foreign = "#!/bin/sh\necho not commitin\n"
	if err := os.WriteFile(path, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Install(gitDir); err == nil {
		t.Fatal("Install overwrote a non-CommitIn hook without error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != foreign {
		t.Fatal("Install modified a non-CommitIn hook despite erroring")
	}
}

func TestUninstallRemovesOwnHook(t *testing.T) {
	gitDir := t.TempDir()
	if err := Install(gitDir); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, err := Uninstall(gitDir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !removed {
		t.Fatal("Uninstall reported removed=false for our own hook")
	}
	if _, err := os.Stat(Path(gitDir)); !os.IsNotExist(err) {
		t.Fatalf("hook file still exists after Uninstall: err=%v", err)
	}
}

func TestUninstallNoopWhenNothingInstalled(t *testing.T) {
	gitDir := t.TempDir()

	removed, err := Uninstall(gitDir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if removed {
		t.Fatal("Uninstall reported removed=true with nothing installed")
	}
}

func TestUninstallRefusesForeignHook(t *testing.T) {
	gitDir := t.TempDir()
	path := Path(gitDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const foreign = "#!/bin/sh\necho not commitin\n"
	if err := os.WriteFile(path, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	removed, err := Uninstall(gitDir)
	if err == nil {
		t.Fatal("Uninstall removed a non-CommitIn hook without error")
	}
	if removed {
		t.Fatal("Uninstall reported removed=true for a non-CommitIn hook")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("foreign hook was deleted: %v", err)
	}
}
