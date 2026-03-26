package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"packrat/internal/archive"
	"packrat/internal/categories"
	"packrat/internal/packages"
	"packrat/internal/profile"
	"packrat/internal/tui"
)

var dryRun bool

var backupCmd = &cobra.Command{
	Use:   "backup [archive-path]",
	Short: "Create a backup archive of the home directory",
	Long: `backup walks the home directory and creates a compressed .tar.zst archive.

An interactive wizard lets you choose categories, output path, and options.
Pass --non-interactive to skip the wizard and use your saved profile.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBackup,
}

func init() {
	backupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Count files without writing the archive")
}

func runBackup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect current user.
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("detect user: %w", err)
	}
	home := u.HomeDir

	// Detect which categories are present on this system.
	allCats := categories.All()
	allCatsWithDirs := categories.WithHomeDirs(allCats, home)

	var detected []categories.Category
	for _, c := range allCatsWithDirs {
		if c.AutoDetect(home) {
			detected = append(detected, c)
		}
	}

	// Detect available package managers.
	detectedManagers := packages.Detect()

	// Default archive path.
	defaultArchivePath := defaultArchiveName(u.Username)
	if len(args) > 0 {
		defaultArchivePath = args[0]
	}

	var selectedIDs []string
	var archivePath string
	var isDryRun bool
	var exportPackages bool
	var selectedManagers []string

	if NonInteractive {
		p, err := profile.Load()
		if err != nil {
			return err
		}
		// Collect IDs enabled in profile.
		for id, enabled := range p.Categories {
			if enabled {
				selectedIDs = append(selectedIDs, id)
			}
		}
		archivePath = defaultArchivePath
		if p.OutputDir != "" && p.OutputDir != "." {
			archivePath = filepath.Join(p.OutputDir, filepath.Base(defaultArchivePath))
		}
		isDryRun = dryRun
		exportPackages = p.ExportPackages
		selectedManagers = p.PackageManagers
	} else {
		wizard, err := tui.RunBackupWizard(detected, defaultArchivePath, detectedManagers)
		if err != nil {
			return fmt.Errorf("backup wizard: %w", err)
		}
		selectedIDs = wizard.SelectedCategories
		archivePath = wizard.ArchivePath
		isDryRun = wizard.DryRun || dryRun
		exportPackages = wizard.ExportPackages
		selectedManagers = wizard.SelectedManagers
	}

	// Filter categories to only those selected.
	selectedSet := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selectedSet[id] = true
	}
	var selectedCats []categories.Category
	for _, c := range allCatsWithDirs {
		if selectedSet[c.ID] {
			selectedCats = append(selectedCats, c)
		}
	}

	cfg := archive.BackupConfig{
		Home:        home,
		Username:    u.Username,
		Categories:  selectedCats,
		ArchivePath: archivePath,
		DryRun:      isDryRun,
	}

	if isDryRun {
		fmt.Println(tui.Title.Render("Dry run — counting files..."))
		result, err := archive.Backup(ctx, cfg)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", tui.Success.Render(fmt.Sprintf("Would archive %d files.", result.FilesArchived)))
		return nil
	}

	// Pre-count files for determinate progress bar.
	fmt.Println(tui.Muted.Render("  Counting files..."))
	countResult, err := archive.Backup(ctx, archive.BackupConfig{
		Home:       cfg.Home,
		Username:   cfg.Username,
		Categories: cfg.Categories,
		DryRun:     true,
	})
	if err != nil {
		return err
	}

	var result archive.BackupResult
	backupStart := time.Now()
	err = tui.RunProgress(tui.BackupProgress{
		Title:      "Backing up",
		TotalFiles: countResult.FilesArchived,
	}, func(progress func(int, string)) error {
		cfg.Progress = progress
		result, err = archive.Backup(ctx, cfg)
		return err
	})
	if err != nil {
		return err
	}
	elapsed := time.Since(backupStart)

	// Export packages if requested.
	pkgCounts := make(map[string]int)
	if exportPackages && len(selectedManagers) > 0 {
		mgrList := make([]packages.Manager, 0, len(selectedManagers))
		for _, m := range selectedManagers {
			mgrList = append(mgrList, packages.Manager(m))
		}
		snap, snapErr := packages.Export(mgrList)
		if snapErr != nil {
			fmt.Fprintf(os.Stderr, "warning: package export failed: %v\n", snapErr)
		} else {
			pkgPath := strings.TrimSuffix(archivePath, ".tar.zst") + ".packages.json"
			if saveErr := packages.SaveSnapshot(pkgPath, snap); saveErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save package snapshot: %v\n", saveErr)
			} else {
				for name, pkgs := range snap.Managers {
					pkgCounts[name] = len(pkgs)
				}
			}
		}
	}

	// Show animated done screen.
	doneStats := tui.DoneStats{
		Files:    result.FilesArchived,
		Size:     result.BytesWritten,
		Elapsed:  elapsed,
		Archive:  result.ArchivePath,
		Packages: pkgCounts,
	}
	if len(pkgCounts) == 0 {
		doneStats.Packages = nil
	}
	if showErr := tui.ShowDoneScreen(doneStats); showErr != nil {
		// Fallback to simple print if done screen fails.
		fmt.Printf("\n%s\n", tui.Success.Render("Backup complete!"))
		fmt.Printf("  Archive: %s\n", result.ArchivePath)
	}

	// Save / update profile.
	p, loadErr := profile.Load()
	if loadErr != nil {
		p = profile.DefaultProfile(u.Username, home)
	}
	// Update enabled categories from this run.
	if p.Categories == nil {
		p.Categories = make(map[string]bool)
	}
	// Reset all to false, then enable selected.
	for _, c := range allCats {
		p.Categories[c.ID] = false
	}
	for _, id := range selectedIDs {
		p.Categories[id] = true
	}
	p.LastArchive = result.ArchivePath
	p.Username = u.Username
	p.Home = home
	p.ExportPackages = exportPackages
	p.PackageManagers = selectedManagers
	if err := profile.Save(p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save profile: %v\n", err)
	}

	return nil
}

// defaultArchiveName returns the default archive filename for the given username.
func defaultArchiveName(username string) string {
	return fmt.Sprintf("%s-packrat-%s.tar.zst", username, time.Now().Format("20060102"))
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatFiles formats a file count with comma separators.
func formatFiles(n int) string {
	s := fmt.Sprintf("%d", n)
	// Insert commas every 3 digits from the right.
	if len(s) <= 3 {
		return s
	}
	result := make([]byte, 0, len(s)+len(s)/3)
	offset := len(s) % 3
	if offset == 0 {
		offset = 3
	}
	result = append(result, s[:offset]...)
	for i := offset; i < len(s); i += 3 {
		result = append(result, ',')
		result = append(result, s[i:i+3]...)
	}
	return string(result)
}
