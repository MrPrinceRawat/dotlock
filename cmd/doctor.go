package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/mrprincerawat/dotlock/internal/crypto"
	"github.com/mrprincerawat/dotlock/internal/envfile"
	"github.com/mrprincerawat/dotlock/internal/gitsetup"
	"github.com/mrprincerawat/dotlock/internal/keystore"
	"github.com/mrprincerawat/dotlock/internal/prompt"
	"github.com/mrprincerawat/dotlock/internal/ui"
	"github.com/mrprincerawat/dotlock/internal/vault"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose the health of the dotlock setup",
	Long:  "Runs diagnostic checks and reports the status of your dotlock configuration.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	ui.Headerf("dotlock doctor")

	// Check 1: .dotlock file exists
	vaultPath := filepath.Join(dir, ".dotlock")
	v, err := vault.Load(vaultPath)
	if err != nil {
		ui.Error("No .dotlock vault found — run 'dotlock init'")
		return nil
	}
	ui.Success(".dotlock vault file exists")

	// Check 2: All env files are encrypted
	envFiles, _ := envfile.DetectEnvFiles(dir)
	vaultEnvs := v.ListEnvs()
	vaultEnvSet := make(map[string]bool)
	for _, e := range vaultEnvs {
		vaultEnvSet[e] = true
	}

	for _, f := range envFiles {
		name := envfile.EnvName(f)
		if !vaultEnvSet[name] {
			ui.Warningf("Env file %s is not encrypted in vault", filepath.Base(f))
		}
	}
	if len(envFiles) > 0 {
		allEncrypted := true
		for _, f := range envFiles {
			if !vaultEnvSet[envfile.EnvName(f)] {
				allEncrypted = false
			}
		}
		if allEncrypted {
			ui.Success("All .env files are encrypted")
		}
	}

	// Check 3: .env.example up to date
	examplePath := filepath.Join(dir, ".env.example")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		ui.Warning(".env.example not found")
	} else {
		// Check if it has all keys from default env
		defaultEnvPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(defaultEnvPath); err == nil {
			envEntries, _ := envfile.Parse(defaultEnvPath)
			exEntries, _ := envfile.Parse(examplePath)
			exKeys := make(map[string]bool)
			for _, e := range exEntries {
				if e.Key != "" {
					exKeys[e.Key] = true
				}
			}
			missing := false
			for _, e := range envEntries {
				if e.Key != "" && !exKeys[e.Key] {
					ui.Warningf(".env.example is missing key: %s", e.Key)
					missing = true
				}
			}
			if !missing {
				ui.Success(".env.example is up to date")
			}
		} else {
			ui.Success(".env.example exists")
		}
	}

	// Check 4: Git protection
	if gitsetup.HasGit(dir) {
		// Check .gitignore
		gitignorePath := filepath.Join(dir, ".gitignore")
		if content, err := os.ReadFile(gitignorePath); err == nil {
			if containsStr(string(content), "# dotlock") {
				ui.Success(".gitignore configured")
			} else {
				ui.Warning(".gitignore missing dotlock patterns")
			}
		} else {
			ui.Error(".gitignore not found")
		}

		// Check hook
		if gitsetup.IsHookInstalled(dir) {
			ui.Success("Pre-commit hook installed")
		} else {
			ui.Error("Pre-commit hook not installed")
		}
	} else {
		ui.Warning("No .git/ directory found")
	}

	// Check 5: Environment key mismatch
	salt, _ := v.GetSalt()
	key, _, keyErr := keystore.ResolveKey(dir, salt, crypto.DeriveKey, func() (string, error) {
		return prompt.Passphrase("Enter passphrase to check env consistency")
	})

	if keyErr == nil && key != nil && len(vaultEnvs) > 1 {
		allKeys := make(map[string]map[string]bool)
		for _, name := range vaultEnvs {
			ct, _ := v.GetEnv(name)
			if plaintext, err := crypto.Decrypt(ct, key); err == nil {
				entries := envfile.ParseString(string(plaintext))
				keys := make(map[string]bool)
				for _, e := range entries {
					if e.Key != "" {
						keys[e.Key] = true
					}
				}
				allKeys[name] = keys
			}
		}

		// Compare all pairs
		sort.Strings(vaultEnvs)
		mismatches := false
		for i := 0; i < len(vaultEnvs); i++ {
			for j := i + 1; j < len(vaultEnvs); j++ {
				e1, e2 := vaultEnvs[i], vaultEnvs[j]
				k1, k2 := allKeys[e1], allKeys[e2]
				for k := range k1 {
					if !k2[k] {
						ui.Warningf("Key %q in %s but not in %s", k, e1, e2)
						mismatches = true
					}
				}
				for k := range k2 {
					if !k1[k] {
						ui.Warningf("Key %q in %s but not in %s", k, e2, e1)
						mismatches = true
					}
				}
			}
		}
		if !mismatches {
			ui.Success("All environments have matching keys")
		}
	}

	return nil
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
