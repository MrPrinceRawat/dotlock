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

var diffCmd = &cobra.Command{
	Use:   "diff [env1] [env2]",
	Short: "Compare environment variables between two environments",
	Long:  "Decrypts and compares two environments, showing added, removed, and changed variables.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
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

	salt, err := v.GetSalt()
	if err != nil {
		return err
	}

	key, _, err := keystore.ResolveKey(dir, salt, crypto.DeriveKey, func() (string, error) {
		return prompt.Passphrase("Enter passphrase")
	})
	if err != nil {
		return err
	}

	var env1Map, env2Map map[string]string
	var label1, label2 string

	if len(args) == 2 {
		// Diff two vault environments
		label1, label2 = args[0], args[1]
		env1Map, err = decryptEnvToMap(v, key, args[0])
		if err != nil {
			return err
		}
		env2Map, err = decryptEnvToMap(v, key, args[1])
		if err != nil {
			return err
		}
	} else {
		// Diff vault env against local file
		envName := args[0]
		label1 = envName + " (vault)"
		label2 = envName + " (local)"

		env1Map, err = decryptEnvToMap(v, key, envName)
		if err != nil {
			return err
		}

		localPath := filepath.Join(dir, envfile.EnvFileName(envName))
		entries, err := envfile.Parse(localPath)
		if err != nil {
			return fmt.Errorf("reading local %s: %w", envfile.EnvFileName(envName), err)
		}
		env2Map = entriesToMap(entries)
	}

	ui.Headerf("Diff: %s ↔ %s", label1, label2)

	// Collect all keys
	allKeys := make(map[string]bool)
	for k := range env1Map {
		allKeys[k] = true
	}
	for k := range env2Map {
		allKeys[k] = true
	}

	sorted := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	changes := 0
	for _, k := range sorted {
		v1, in1 := env1Map[k]
		v2, in2 := env2Map[k]

		if in1 && !in2 {
			ui.DiffRemove(fmt.Sprintf("%s=%s", k, v1))
			changes++
		} else if !in1 && in2 {
			ui.DiffAdd(fmt.Sprintf("%s=%s", k, v2))
			changes++
		} else if v1 != v2 {
			ui.DiffChange(fmt.Sprintf("%s: %q → %q", k, v1, v2))
			changes++
		}
	}

	if changes == 0 {
		ui.Success("No differences found.")
	} else {
		fmt.Fprintf(os.Stderr, "\n%d difference(s) found.\n", changes)
	}

	return nil
}

func decryptEnvToMap(v *vault.Vault, key []byte, name string) (map[string]string, error) {
	ct, ok := v.GetEnv(name)
	if !ok {
		return nil, fmt.Errorf("environment %q not found in vault", name)
	}

	plaintext, err := crypto.Decrypt(ct, key)
	if err != nil {
		return nil, fmt.Errorf("wrong passphrase")
	}

	entries := envfile.ParseString(string(plaintext))
	return entriesToMap(entries), nil
}

func entriesToMap(entries []envfile.Entry) map[string]string {
	m := make(map[string]string)
	for _, e := range entries {
		if e.Key != "" {
			m[e.Key] = e.Value
		}
	}
	return m
}
