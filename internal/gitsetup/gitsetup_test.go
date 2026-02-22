package gitsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitDir(t *testing.T) string {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755)
	os.MkdirAll(filepath.Join(dir, ".git", "info"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte("# default excludes\n"), 0644)
	return dir
}

func TestSetupGitignoreCreate(t *testing.T) {
	dir := t.TempDir()
	if err := SetupGitignore(dir); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(content), "# dotlock") {
		t.Fatal("expected dotlock header")
	}
	if !strings.Contains(string(content), ".env*") {
		t.Fatal("expected .env* pattern")
	}
}

func TestSetupGitignoreAppend(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)

	if err := SetupGitignore(dir); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(content), "node_modules/") {
		t.Fatal("should preserve existing content")
	}
	if !strings.Contains(string(content), "# dotlock") {
		t.Fatal("should append dotlock patterns")
	}
}

func TestSetupGitignoreSkipDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("# dotlock\n.env*\n"), 0644)

	if err := SetupGitignore(dir); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	count := strings.Count(string(content), "# dotlock")
	if count != 1 {
		t.Fatalf("expected 1 occurrence, got %d", count)
	}
}

func TestInstallHookCreate(t *testing.T) {
	dir := setupGitDir(t)
	if err := InstallHook(dir); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	content, _ := os.ReadFile(hookPath)
	if !strings.Contains(string(content), "#!/bin/sh") {
		t.Fatal("expected shebang")
	}
	if !strings.Contains(string(content), dotlockMarker) {
		t.Fatal("expected dotlock marker")
	}

	info, _ := os.Stat(hookPath)
	if info.Mode().Perm()&0100 == 0 {
		t.Fatal("hook should be executable")
	}
}

func TestInstallHookAppend(t *testing.T) {
	dir := setupGitDir(t)
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'existing hook'\n"), 0755)

	if err := InstallHook(dir); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(hookPath)
	if !strings.Contains(string(content), "existing hook") {
		t.Fatal("should preserve existing hook")
	}
	if !strings.Contains(string(content), dotlockMarker) {
		t.Fatal("should append dotlock hook")
	}
}

func TestInstallHookSkipDuplicate(t *testing.T) {
	dir := setupGitDir(t)
	if err := InstallHook(dir); err != nil {
		t.Fatal(err)
	}
	if err := InstallHook(dir); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-commit"))
	count := strings.Count(string(content), dotlockMarker)
	if count != 1 {
		t.Fatalf("expected 1 marker, got %d", count)
	}
}

func TestHasGit(t *testing.T) {
	dir := setupGitDir(t)
	if !HasGit(dir) {
		t.Fatal("expected git")
	}
	if HasGit(t.TempDir()) {
		t.Fatal("expected no git")
	}
}

func TestLazySetup(t *testing.T) {
	dir := setupGitDir(t)
	did, err := LazySetup(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !did {
		t.Fatal("expected setup to run")
	}

	// Second call should be no-op
	did, err = LazySetup(dir)
	if err != nil {
		t.Fatal(err)
	}
	if did {
		t.Fatal("expected no-op on second call")
	}
}
