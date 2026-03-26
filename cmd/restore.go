package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"packrat/internal/archive"
	"packrat/internal/profile"
	"packrat/internal/tui"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <archive-path> [destination]",
	Short: "Restore a backup archive",
	Long: `restore extracts a .tar.zst archive to the specified destination.

If destination is omitted it defaults to "/" (restoring files to their
original absolute paths). Restoring to "/" requires root privileges.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runRestore,
}

func runRestore(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	var archivePath, destination string

	switch {
	case len(args) >= 2:
		archivePath = args[0]
		destination = args[1]

	case len(args) == 1:
		archivePath = args[0]
		destination = "/"

	case NonInteractive:
		p, err := profile.Load()
		if err != nil {
			return err
		}
		if p.LastArchive == "" {
			return fmt.Errorf("no last archive recorded in profile; provide an archive path explicitly")
		}
		archivePath = p.LastArchive
		destination = "/"

	default:
		res, err := tui.RunRestoreWizard()
		if err != nil {
			return fmt.Errorf("restore wizard: %w", err)
		}
		archivePath = res.ArchivePath
		destination = res.Destination
	}

	// Warn if restoring to "/" without root.
	if destination == "/" && os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, tui.Error.Render(
			"Warning: restoring to / without root privileges may fail for some files.",
		))
	}

	err := tui.RunProgress(tui.BackupProgress{
		Title:      "Restoring",
		TotalFiles: 0, // indeterminate — pulse the bar
	}, func(progress func(int, string)) error {
		return archive.Restore(ctx, archivePath, destination, progress)
	})
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n", tui.Success.Render("Restore complete!"))
	fmt.Printf("  Archive:     %s\n", archivePath)
	fmt.Printf("  Destination: %s\n", destination)

	// Check for a .packages.json sidecar file.
	pkgPath := strings.TrimSuffix(archivePath, ".tar.zst") + ".packages.json"
	if _, err := os.Stat(pkgPath); err == nil {
		fmt.Printf("\n  %s %s\n", tui.Accent.Render("Package snapshot found:"), pkgPath)
		fmt.Printf("  Run: %s\n", tui.Bold.Render("packrat packages restore "+pkgPath))
	}
	return nil
}
