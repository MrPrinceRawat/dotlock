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

var unlockCmd = &cobra.Command{
	Use:   "unlock [env]",
	Short: "Decrypt environments from the .dotlock vault into .env files",
	Long:  "Decrypts all environments (or a specific one) from the .dotlock vault into .env files.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUnlock,
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}

func runUnlock(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	vaultPath := filepath.Join(dir, ".dotlock")

	v, err := vault.Load(vaultPath)
	if err != nil {
		ui.Error("No .dotlock vault found. Run 'dotlock init' first.")
		return fmt.Errorf("loading vault: %w", err)
	}

	salt, err := v.GetSalt()
	if err != nil {
		return fmt.Errorf("reading salt: %w", err)
	}

	key, _, err := keystore.ResolveKey(dir, salt, crypto.DeriveKey, func() (string, error) {
		return prompt.Passphrase("Enter passphrase")
	})
	if err != nil {
		return err
	}

	// Lazy git setup
	if did, err := gitsetup.LazySetup(dir); err != nil {
		ui.Warningf("Git setup issue: %v", err)
	} else if did {
		ui.Success("Auto-installed git protection")
	}

	// Determine which envs to unlock
	var envNames []string
	if len(args) > 0 {
		envNames = []string{args[0]}
	} else {
		envNames = v.ListEnvs()
	}

	if len(envNames) == 0 {
		ui.Warning("No environments in vault.")
		return nil
	}

	for _, name := range envNames {
		ct, ok := v.GetEnv(name)
		if !ok {
			ui.Warningf("Environment %q not found in vault", name)
			continue
		}

		plaintext, err := crypto.Decrypt(ct, key)
		if err != nil {
			ui.Error("Wrong passphrase.")
			return fmt.Errorf("wrong passphrase")
		}

		fileName := envfile.EnvFileName(name)
		outPath := filepath.Join(dir, fileName)
		if err := os.WriteFile(outPath, plaintext, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", fileName, err)
		}
		ui.Successf("Unlocked %s → %s", name, fileName)
	}

	return nil
}
