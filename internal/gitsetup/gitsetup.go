package gitsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const dotlockMarker = "# --- dotlock pre-commit hook ---"

const gitignorePatterns = `# dotlock
.env*
!.env.example
!.env.template
!.env.sample
.dotlock.key
`

const hookScript = `
# --- dotlock pre-commit hook ---
# Block .env files from being committed
BLOCKED_FILES=$(git diff --cached --name-only | grep -E '^\.env' | grep -v '\.env\.example$' | grep -v '\.env\.template$' | grep -v '\.env\.sample$' || true)
if [ -n "$BLOCKED_FILES" ]; then
  echo "dotlock: Blocked .env files from commit:"
  echo "$BLOCKED_FILES" | while read f; do echo "  - $f"; done
  echo ""
  echo "Run 'git reset HEAD <file>' to unstage, or use 'dotlock lock' to encrypt."
  exit 1
fi

# Auto-lock if dotlock is installed and .env files have changed
if command -v dotlock &> /dev/null && [ -f ".dotlock" ]; then
  ENV_CHANGED=false
  for f in .env .env.*; do
    if [ -f "$f" ] && [[ "$f" != *.example ]] && [[ "$f" != *.template ]] && [[ "$f" != *.sample ]]; then
      ENV_CHANGED=true
      break
    fi
  done
  if [ "$ENV_CHANGED" = true ]; then
    dotlock lock 2>/dev/null
    git add .dotlock .env.example 2>/dev/null || true
  fi
fi

# Scan staged content for secrets
STAGED_CONTENT=$(git diff --cached --diff-filter=ACM -U0 | grep -E '^\+' | grep -v '^\+\+\+' || true)
if echo "$STAGED_CONTENT" | grep -qE '(sk_live_|sk_test_|AKIA[0-9A-Z]{16}|ghp_[a-zA-Z0-9]{36}|postgres://[^:]+:[^@]+@|mysql://[^:]+:[^@]+@|redis://[^:]+:[^@]+@)'; then
  echo "dotlock: WARNING - Possible secrets detected in staged changes!"
  echo "Review your staged files before committing."
fi
# --- end dotlock pre-commit hook ---
`

// HasGit returns true if .git/ exists in the given directory.
func HasGit(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// SetupGitignore appends dotlock patterns to .gitignore, creating if needed.
func SetupGitignore(dir string) error {
	path := filepath.Join(dir, ".gitignore")

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	// Skip if already configured
	if strings.Contains(string(content), "# dotlock") {
		return nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer f.Close()

	// Add newline separator if file has content
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n\n") {
		fmt.Fprintln(f)
	}
	fmt.Fprint(f, gitignorePatterns)
	return nil
}

// SetupExclude appends dotlock patterns to .git/info/exclude.
func SetupExclude(dir string) error {
	excludePath := filepath.Join(dir, ".git", "info", "exclude")

	content, err := os.ReadFile(excludePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no .git/info/exclude, skip
		}
		return fmt.Errorf("reading exclude: %w", err)
	}

	if strings.Contains(string(content), "# dotlock") {
		return nil
	}

	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening exclude: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f)
	fmt.Fprint(f, gitignorePatterns)
	return nil
}

// GenerateHookScript returns the pre-commit hook script content.
func GenerateHookScript() string {
	return hookScript
}

// InstallHook installs the dotlock pre-commit hook.
func InstallHook(dir string) error {
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")

	// Ensure hooks directory exists
	hooksDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks dir: %w", err)
	}

	content, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading pre-commit hook: %w", err)
	}

	// Skip if already installed
	if strings.Contains(string(content), dotlockMarker) {
		return nil
	}

	f, err := os.OpenFile(hookPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("opening pre-commit hook: %w", err)
	}
	defer f.Close()

	// Add shebang if creating new file
	if len(content) == 0 {
		fmt.Fprintln(f, "#!/bin/sh")
	}

	fmt.Fprint(f, hookScript)

	// Ensure executable
	return os.Chmod(hookPath, 0755)
}

// IsHookInstalled checks if the dotlock hook marker is present.
func IsHookInstalled(dir string) bool {
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), dotlockMarker)
}

// LazySetup checks for .git/ and installs protection if needed.
// Returns true if setup was performed, false if already done or no git.
func LazySetup(dir string) (bool, error) {
	if !HasGit(dir) {
		return false, nil
	}

	if IsHookInstalled(dir) {
		return false, nil
	}

	if err := SetupGitignore(dir); err != nil {
		return false, err
	}
	if err := SetupExclude(dir); err != nil {
		return false, err
	}
	if err := InstallHook(dir); err != nil {
		return false, err
	}

	return true, nil
}
