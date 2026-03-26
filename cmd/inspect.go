package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"packrat/internal/archive"
	"packrat/internal/packages"
	"packrat/internal/tui"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <archive-path>",
	Short: "Show the contents of a backup archive",
	Long: `inspect reads the archive headers and prints a summary grouped by
top-level path prefix, including entry count and total size per group.`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func runInspect(cmd *cobra.Command, args []string) error {
	archivePath := args[0]

	entries, err := archive.Inspect(archivePath)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}

	fmt.Println(tui.Title.Render("Archive: " + archivePath))

	if err := tui.RunInspectTable(entries); err != nil {
		return fmt.Errorf("inspect table: %w", err)
	}

	// Check for a .packages.json sidecar.
	pkgPath := strings.TrimSuffix(archivePath, ".tar.zst") + ".packages.json"
	if _, err := os.Stat(pkgPath); err == nil {
		snap, snapErr := packages.LoadSnapshot(pkgPath)
		if snapErr == nil {
			fmt.Printf("\n%s\n", tui.Accent.Render("Package snapshot: "+pkgPath))
			for mgr, pkgs := range snap.Managers {
				fmt.Printf("  %s: %d packages\n", tui.Bold.Render(mgr), len(pkgs))
			}
		}
	}
	return nil
}
