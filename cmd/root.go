// Package cmd implements the packrat command-line interface.
package cmd

import (
	"github.com/spf13/cobra"
)

// NonInteractive is set to true when --non-interactive / -y is passed.
// Subcommands read this flag to skip TUI prompts and load the saved profile.
var NonInteractive bool

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "packrat",
	Short: "Backup and restore WSL home directories",
	Long: `packrat creates and restores compressed archives of your WSL home directory.

Use the interactive wizard (default) or pass --non-interactive to use your
saved profile from ~/.config/packrat/profile.json.`,
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&NonInteractive, "non-interactive", "y", false,
		"Skip prompts and use saved profile (~/.config/packrat/profile.json)")

	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(packagesCmd)
}
