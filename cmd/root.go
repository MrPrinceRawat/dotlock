package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "dotlock",
	Short: "Encrypt and manage .env files with a passphrase",
	Long:  "dotlock encrypts your .env files into a committable .dotlock vault file, protected by a shared passphrase. No cloud, no accounts, no SaaS.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
}
