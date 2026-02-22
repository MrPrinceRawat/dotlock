package cmd

import (
	"fmt"
	"os"

	"github.com/mrprincerawat/dotlock/internal/scanner"
	"github.com/mrprincerawat/dotlock/internal/ui"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the codebase for hardcoded secrets",
	Long:  "Walks the project files and reports any patterns matching common secret formats.",
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	ui.Headerf("Scanning for hardcoded secrets...")

	findings, err := scanner.ScanDir(dir)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	if len(findings) == 0 {
		ui.Success("No hardcoded secrets found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nFound %d potential secret(s):\n\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stdout, "  %s%s:%d%s  [%s%s%s]\n",
			ui.Cyan, f.File, f.Line, ui.Reset,
			ui.Yellow, f.Type, ui.Reset)
		fmt.Fprintf(os.Stdout, "    %s\n\n", f.Content)
	}

	ui.Warningf("Found %d potential secret(s). Consider moving them to .env files.", len(findings))

	return nil
}
