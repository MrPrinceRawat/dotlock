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

var lockCmd = &cobra.Command{
	Use:   "lock [env]",
	Short: "Encrypt .env files into the .dotlock vault",
	Long:  "Encrypts all detected .env files (or a specific environment) into the .dotlock vault and regenerates .env.example.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLock,
}

func init() {
	rootCmd.AddCommand(lockCmd)
}

func runLock(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	vaultPath := filepath.Join(dir, ".dotlock")

	// Load existing vault
	v, err := vault.Load(vaultPath)
	if err != nil {
		ui.Error("No .dotlock vault found. Run 'dotlock init' first.")
		return fmt.Errorf("loading vault: %w", err)
	}

	// Resolve key
	salt, err := v.GetSalt()
	if err != nil {
		return fmt.Errorf("reading salt: %w", err)
	}

	key, source, err := keystore.ResolveKey(dir, salt, crypto.DeriveKey, func() (string, error) {
		return prompt.Passphrase("Enter passphrase")
	})
	if err != nil {
		return err
	}

	// Verify key by trying to decrypt an existing env
	if err := verifyKey(v, key); err != nil {
		return err
	}

	_ = source

	// Lazy git setup
	if did, err := gitsetup.LazySetup(dir); err != nil {
		ui.Warningf("Git setup issue: %v", err)
	} else if did {
		ui.Success("Auto-installed git protection")
	}

	// Determine which files to lock
	var envFiles []string
	if len(args) > 0 {
		envName := args[0]
		fileName := envfile.EnvFileName(envName)
		envPath := filepath.Join(dir, fileName)
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			return fmt.Errorf("env file %s not found", fileName)
		}
		envFiles = []string{envPath}
	} else {
		envFiles, err = envfile.DetectEnvFiles(dir)
		if err != nil {
			return err
		}
	}

	if len(envFiles) == 0 {
		ui.Error("No .env files to lock.")
		return fmt.Errorf("no .env files found")
	}

	// Encrypt each file
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
		ui.Successf("Locked %s → %s", filepath.Base(envPath), name)
	}

	// Save vault
	if err := v.Save(vaultPath); err != nil {
		return fmt.Errorf("saving vault: %w", err)
	}

	// Regenerate .env.example
	if err := generateEnvExample(dir, envFiles); err != nil {
		ui.Warningf("Could not regenerate .env.example: %v", err)
	} else {
		ui.Success("Regenerated .env.example")
	}

	return nil
}

// verifyKey tries to decrypt any existing environment to check if the key is correct.
func verifyKey(v *vault.Vault, key []byte) error {
	envs := v.ListEnvs()
	if len(envs) == 0 {
		return nil // no existing envs to verify against
	}

	ct, _ := v.GetEnv(envs[0])
	_, err := crypto.Decrypt(ct, key)
	if err != nil {
		ui.Error("Wrong passphrase.")
		return fmt.Errorf("wrong passphrase")
	}
	return nil
}
