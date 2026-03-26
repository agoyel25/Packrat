package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"packrat/internal/categories"
	"packrat/internal/packages"
)

// WizardResult holds the selections made during the backup wizard.
type WizardResult struct {
	// SelectedCategories contains the IDs of categories the user enabled.
	SelectedCategories []string
	// ArchivePath is the output file path chosen by the user.
	ArchivePath string
	// DryRun indicates whether to simulate without writing.
	DryRun bool
	// ExportPackages indicates whether to export installed packages.
	ExportPackages bool
	// SelectedManagers holds the manager names chosen for package export.
	SelectedManagers []string
}

// RestoreResult holds the selections made during the restore wizard.
type RestoreResult struct {
	// ArchivePath is the archive file to restore.
	ArchivePath string
	// Destination is the root directory to extract into.
	Destination string
}

// stepIndicator returns a string like "Step 2 of 4  ●●○○"
func stepIndicator(current, total int) string {
	var dots strings.Builder
	for i := 1; i <= total; i++ {
		if i <= current {
			dots.WriteString(Accent.Render("●"))
		} else {
			dots.WriteString(Muted.Render("○"))
		}
	}
	return fmt.Sprintf("%s  %s", Muted.Render(fmt.Sprintf("Step %d of %d", current, total)), dots.String())
}

// RunBackupWizard presents the interactive backup wizard and returns the user's
// choices. Only categories present in detectedCategories are shown.
func RunBackupWizard(detectedCategories []categories.Category, defaultArchivePath string, detectedManagers []packages.Manager) (WizardResult, error) {
	if len(detectedCategories) == 0 {
		return WizardResult{}, fmt.Errorf("no backup categories detected on this system")
	}

	fmt.Println(Banner())

	// Build huh multi-select options with colored labels.
	options := make([]huh.Option[string], len(detectedCategories))
	selectedIDs := make([]string, len(detectedCategories))
	for i, c := range detectedCategories {
		color := CategoryColors[c.ID]
		icon := CategoryIcon(c.ID)
		label := color.Render(icon + " " + c.Label)
		options[i] = huh.NewOption(
			fmt.Sprintf("%s  %s", label, Muted.Render(c.Description)),
			c.ID,
		)
		selectedIDs[i] = c.ID // all selected by default
	}

	var result WizardResult
	result.SelectedCategories = selectedIDs
	result.ArchivePath = defaultArchivePath

	hasManagers := len(detectedManagers) > 0
	totalSteps := 3
	if hasManagers {
		totalSteps = 4
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewNote().Title(stepIndicator(1, totalSteps)).Description(""),
			huh.NewMultiSelect[string]().
				Title(Title.Render("Select categories to back up")).
				Description(Muted.Render("Space to toggle, Enter to confirm")).
				Options(options...).
				Value(&result.SelectedCategories),
		),
		huh.NewGroup(
			huh.NewNote().Title(stepIndicator(2, totalSteps)).Description(""),
			huh.NewInput().
				Title(Title.Render("Output archive path")).
				Description(Muted.Render("Path for the .tar.zst archive file")).
				Value(&result.ArchivePath).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("path cannot be empty")
					}
					return nil
				}),
		),
	}

	if hasManagers {
		// Build manager options.
		mgrOptions := make([]huh.Option[string], len(detectedManagers))
		for i, m := range detectedManagers {
			mgrOptions[i] = huh.NewOption(string(m), string(m))
			result.SelectedManagers = append(result.SelectedManagers, string(m))
		}

		groups = append(groups, huh.NewGroup(
			huh.NewNote().Title(stepIndicator(3, totalSteps)).Description(""),
			huh.NewConfirm().
				Title(Title.Render("Export installed packages?")).
				Description(Muted.Render("Save a list of installed packages alongside the archive")).
				Value(&result.ExportPackages),
			huh.NewMultiSelect[string]().
				Title(Title.Render("Select package managers to snapshot")).
				Options(mgrOptions...).
				Value(&result.SelectedManagers),
		))
	}

	dryRunStep := totalSteps
	groups = append(groups, huh.NewGroup(
		huh.NewNote().Title(stepIndicator(dryRunStep, totalSteps)).Description(""),
		huh.NewConfirm().
			Title(Title.Render("Dry run?")).
			Description(Muted.Render("Count files without writing the archive")).
			Value(&result.DryRun),
	))

	form := huh.NewForm(groups...)

	if err := form.Run(); err != nil {
		return WizardResult{}, err
	}

	if len(result.SelectedCategories) == 0 {
		return WizardResult{}, fmt.Errorf("no categories selected")
	}

	return result, nil
}

// RunRestoreWizard presents the interactive restore wizard and returns the
// user's choices.
func RunRestoreWizard() (RestoreResult, error) {
	var result RestoreResult
	result.Destination = "/"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(Title.Render("Archive path")).
				Description(Muted.Render("Path to the .tar.zst archive to restore")).
				Value(&result.ArchivePath).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("archive path cannot be empty")
					}
					if _, err := os.Stat(s); os.IsNotExist(err) {
						return fmt.Errorf("file not found: %s", s)
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title(Title.Render("Destination directory")).
				Description(Muted.Render("Root to extract into (default: /)")).
				Value(&result.Destination).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("destination cannot be empty")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return RestoreResult{}, err
	}

	return result, nil
}
