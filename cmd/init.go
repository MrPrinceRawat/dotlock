package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrprincerawat/dotlock/internal/crypto"
	"github.com/mrprincerawat/dotlock/internal/envfile"
	"github.com/mrprincerawat/dotlock/internal/gitsetup"
	"github.com/mrprincerawat/dotlock/internal/keystore"
	"github.com/mrprincerawat/dotlock/internal/prompt"
	"github.com/mrprincerawat/dotlock/internal/ui"
	"github.com/mrprincerawat/dotlock/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dotlock in the current project",
	Long:  "Auto-detects .env files, encrypts them into a .dotlock vault, sets up git protection, and caches the key.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	vaultPath := filepath.Join(dir, ".dotlock")

	// Check if already initialized
	if _, err := os.Stat(vaultPath); err == nil {
		ui.Error("dotlock is already initialized in this directory.")
		return fmt.Errorf("dotlock already initialized")
	}

	// Detect env files
	envFiles, err := envfile.DetectEnvFiles(dir)
	if err != nil {
		return fmt.Errorf("detecting env files: %w", err)
	}
	if len(envFiles) == 0 {
		ui.Error("No .env files found in this directory.")
		return fmt.Errorf("no .env files found")
	}

	ui.Headerf("dotlock init")
	fmt.Fprintf(os.Stderr, "Found %d env file(s):\n", len(envFiles))
	for _, f := range envFiles {
		fmt.Fprintf(os.Stderr, "  • %s\n", filepath.Base(f))
	}
	fmt.Fprintln(os.Stderr)

	// Get passphrase: env var first, then interactive prompt
	passphrase := os.Getenv("DOTLOCK_PASSPHRASE")
	if passphrase == "" {
		passphrase, err = prompt.PassphraseWithConfirm()
		if err != nil {
			return err
		}
		if passphrase == "" {
			return fmt.Errorf("passphrase cannot be empty")
		}
	}

	// Generate salt and derive key
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	key := crypto.DeriveKey(passphrase, salt)

	// Create vault
	v := vault.New(salt)

	// Encrypt each env file
	for _, envPath := range envFiles {
		content, err := os.ReadFile(envPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", envPath, err)
		}

		encrypted, err := crypto.Encrypt(content, key)
		if err != nil {
			return fmt.Errorf("encrypting %s: %w", envPath, err)
		}

		name := envfile.EnvName(envPath)
		v.SetEnv(name, encrypted)
		ui.Successf("Encrypted %s → %s", filepath.Base(envPath), name)
	}

	// Save vault
	if err := v.Save(vaultPath); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}
	ui.Success("Created .dotlock vault")

	// Cache key
	if err := keystore.SaveKey(dir, key); err != nil {
		ui.Warningf("Could not cache key: %v", err)
	} else {
		ui.Success("Cached encryption key")
	}

	// Generate .env.example from default env
	if err := generateEnvExample(dir, envFiles); err != nil {
		ui.Warningf("Could not generate .env.example: %v", err)
	} else {
		ui.Success("Generated .env.example")
	}

	// Generate .dotlock.readme
	if err := generateDotlockReadme(dir); err != nil {
		ui.Warningf("Could not generate .dotlock.readme: %v", err)
	} else {
		ui.Success("Generated .dotlock.readme")
	}

	// Git setup
	if gitsetup.HasGit(dir) {
		if err := gitsetup.SetupGitignore(dir); err != nil {
			ui.Warningf("Could not configure .gitignore: %v", err)
		} else {
			ui.Success("Configured .gitignore")
		}

		if err := gitsetup.SetupExclude(dir); err != nil {
			ui.Warningf("Could not configure .git/info/exclude: %v", err)
		}

		if err := gitsetup.InstallHook(dir); err != nil {
			ui.Warningf("Could not install pre-commit hook: %v", err)
		} else {
			ui.Success("Installed pre-commit hook")
		}
	} else {
		ui.Warning("No .git/ found — skipping git setup (will auto-install on next command when git is detected)")
	}

	fmt.Fprintln(os.Stderr)
	ui.Headerf("Done! Your .env files are encrypted in .dotlock")
	fmt.Fprintf(os.Stderr, "Share the passphrase with your team via a secure channel.\n")
	fmt.Fprintf(os.Stderr, "Teammates run: %sdotlock unlock%s\n", ui.Bold, ui.Reset)

	return nil
}

func generateEnvExample(dir string, envFiles []string) error {
	// Use the default .env if it exists, otherwise first file
	var targetPath string
	for _, f := range envFiles {
		if envfile.EnvName(f) == "default" {
			targetPath = f
			break
		}
	}
	if targetPath == "" && len(envFiles) > 0 {
		targetPath = envFiles[0]
	}
	if targetPath == "" {
		return nil
	}

	entries, err := envfile.Parse(targetPath)
	if err != nil {
		return err
	}

	example := envfile.GenerateExample(entries)
	return os.WriteFile(filepath.Join(dir, ".env.example"), []byte(example), 0644)
}

func generateDotlockReadme(dir string) error {
	content := `# dotlock

This project uses dotlock to manage environment variables securely.

## First-time setup

1. Get the shared passphrase from your team
2. Run: dotlock unlock
3. Enter the passphrase when prompted

## How it works

- .env files are encrypted into the .dotlock vault file
- The .dotlock file is safe to commit (it's encrypted)
- Your passphrase is cached locally after first use
- A pre-commit hook auto-locks changes on every commit

## Commands

- dotlock unlock    — Decrypt .env files from the vault
- dotlock lock      — Encrypt .env files into the vault
- dotlock ls        — List environments in the vault
- dotlock diff      — Compare environments
- dotlock doctor    — Check setup health
- dotlock scan      — Scan for hardcoded secrets

## CI/CD

Set the DOTLOCK_PASSPHRASE environment variable in your CI/CD pipeline.
`
	return os.WriteFile(filepath.Join(dir, ".dotlock.readme"), []byte(content), 0644)
}
