package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mrprincerawat/dotlock/internal/crypto"
	"github.com/mrprincerawat/dotlock/internal/envfile"
	"github.com/mrprincerawat/dotlock/internal/keystore"
	"github.com/mrprincerawat/dotlock/internal/prompt"
	"github.com/mrprincerawat/dotlock/internal/ui"
	"github.com/mrprincerawat/dotlock/internal/vault"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List environments stored in the vault",
	Long:  "Displays each environment name and the number of variables it contains.",
	RunE:  runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	vaultPath := filepath.Join(dir, ".dotlock")
	v, err := vault.Load(vaultPath)
	if err != nil {
		ui.Error("No .dotlock vault found.")
		return fmt.Errorf("loading vault: %w", err)
	}

	envs := v.ListEnvs()
	if len(envs) == 0 {
		ui.Warning("No environments in vault.")
		return nil
	}

	sort.Strings(envs)

	// Try to get key to show var counts
	salt, _ := v.GetSalt()
	key, _, _ := keystore.ResolveKey(dir, salt, crypto.DeriveKey, func() (string, error) {
		return prompt.Passphrase("Enter passphrase")
	})

	ui.Headerf("Environments in .dotlock")
	for _, name := range envs {
		ct, _ := v.GetEnv(name)
		varCount := "?"
		if key != nil {
			if plaintext, err := crypto.Decrypt(ct, key); err == nil {
				entries := envfile.ParseString(string(plaintext))
				count := 0
				for _, e := range entries {
					if e.Key != "" {
						count++
					}
				}
				varCount = fmt.Sprintf("%d", count)
			}
		}
		fmt.Fprintf(os.Stdout, "  %-20s %s variables\n", name, varCount)
	}

	return nil
}
