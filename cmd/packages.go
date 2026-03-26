package cmd

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"packrat/internal/packages"
	"packrat/internal/tui"
)

var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Manage system package snapshots",
	Long:  `Export and restore system package manager snapshots alongside backups.`,
}

var packagesExportCmd = &cobra.Command{
	Use:   "export [output-path]",
	Short: "Export installed packages to a JSON snapshot",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPackagesExport,
}

var packagesRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-path>",
	Short: "Show restore commands for a package snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackagesRestore,
}

func init() {
	packagesCmd.AddCommand(packagesExportCmd)
	packagesCmd.AddCommand(packagesRestoreCmd)
}

func runPackagesExport(cmd *cobra.Command, args []string) error {
	detected := packages.Detect()
	if len(detected) == 0 {
		return fmt.Errorf("no supported package managers found (apt, pacman, snap)")
	}

	// Determine output path.
	var outputPath string
	if len(args) > 0 {
		outputPath = args[0]
	} else {
		u, err := user.Current()
		if err != nil {
			return fmt.Errorf("detect user: %w", err)
		}
		outputPath = fmt.Sprintf("%s-packages-%s.json", u.Username, time.Now().Format("20060102"))
	}

	var selectedManagers []string
	for _, m := range detected {
		selectedManagers = append(selectedManagers, string(m))
	}

	if !NonInteractive {
		mgrOptions := make([]huh.Option[string], len(detected))
		for i, m := range detected {
			mgrOptions[i] = huh.NewOption(string(m), string(m))
		}
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title(tui.Title.Render("Select package managers to snapshot")).
					Description(tui.Muted.Render("Space to toggle, Enter to confirm")).
					Options(mgrOptions...).
					Value(&selectedManagers),
			),
		)
		if err := form.Run(); err != nil {
			return fmt.Errorf("package selection: %w", err)
		}
	}

	if len(selectedManagers) == 0 {
		return fmt.Errorf("no package managers selected")
	}

	mgrList := make([]packages.Manager, 0, len(selectedManagers))
	for _, m := range selectedManagers {
		mgrList = append(mgrList, packages.Manager(m))
	}

	snap, err := packages.Export(mgrList)
	if err != nil {
		return fmt.Errorf("export packages: %w", err)
	}

	if err := packages.SaveSnapshot(outputPath, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	fmt.Printf("\n%s\n", tui.Success.Render("Package snapshot saved!"))
	fmt.Printf("  Path: %s\n", tui.Bold.Render(outputPath))
	for mgr, pkgs := range snap.Managers {
		fmt.Printf("  %s: %d packages\n", tui.Accent.Render(mgr), len(pkgs))
	}
	return nil
}

func runPackagesRestore(cmd *cobra.Command, args []string) error {
	snapshotPath := args[0]

	snap, err := packages.LoadSnapshot(snapshotPath)
	if err != nil {
		return fmt.Errorf("load snapshot: %w", err)
	}

	fmt.Printf("\n%s\n", tui.Title.Render("Package snapshot: "+snapshotPath))
	fmt.Printf("  Exported at: %s\n\n", tui.Muted.Render(snap.ExportedAt))

	for mgrName, pkgs := range snap.Managers {
		fmt.Printf("%s  (%d packages)\n", tui.Accent.Render(mgrName), len(pkgs))
	}

	fmt.Printf("\n%s\n", tui.Bold.Render("Install commands:"))
	fmt.Println(tui.Muted.Render("Run these commands to restore your packages:"))
	fmt.Println()

	for mgrName, pkgs := range snap.Managers {
		installCmd := packages.InstallCommand(packages.Manager(mgrName), pkgs)
		if installCmd == "" {
			continue
		}
		fmt.Printf("%s\n", tui.Accent.Render(mgrName+":"))
		fmt.Printf("%s\n\n", installCmd)
	}

	fmt.Fprintln(os.Stderr, tui.Muted.Render("Note: commands are not executed automatically. Run them manually with appropriate privileges."))
	return nil
}
