// Package profile handles loading and saving the user's backup preferences.
package profile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"packrat/internal/categories"
)

// ErrNoProfile is returned by Load when no saved profile exists.
var ErrNoProfile = errors.New("no saved profile found; run packrat interactively first")

// configDir returns the path to the packrat config directory.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "packrat"), nil
}

// profilePath returns the full path to profile.json.
func profilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profile.json"), nil
}

// Profile stores the user's saved backup preferences.
type Profile struct {
	// Username is the OS username detected when the profile was created.
	Username string `json:"username"`
	// Home is the user's home directory path.
	Home string `json:"home"`
	// Categories maps category IDs to whether they are enabled.
	Categories map[string]bool `json:"categories"`
	// OutputDir is the default directory for new archives.
	OutputDir string `json:"output_dir"`
	// LastArchive is the path of the most recently created backup archive.
	LastArchive string `json:"last_archive"`
	// ExportPackages indicates whether to export installed packages during backup.
	ExportPackages bool `json:"export_packages"`
	// PackageManagers lists the package manager names to snapshot.
	PackageManagers []string `json:"package_managers"`
}

// Load reads the profile from ~/.config/packrat/profile.json.
// It returns ErrNoProfile if the file does not exist.
func Load() (*Profile, error) {
	path, err := profilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoProfile
		}
		return nil, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save writes the profile to ~/.config/packrat/profile.json, creating
// the directory if necessary.
func Save(p *Profile) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "profile.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// DefaultProfile constructs a Profile with all detected categories enabled.
func DefaultProfile(username, home string) *Profile {
	cats := categories.All()
	enabled := make(map[string]bool, len(cats))
	for _, c := range cats {
		enabled[c.ID] = c.AutoDetect(home)
	}
	return &Profile{
		Username:   username,
		Home:       home,
		Categories: enabled,
		OutputDir:  ".",
	}
}
