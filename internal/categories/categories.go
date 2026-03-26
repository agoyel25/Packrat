// Package categories defines the backup categories and their filesystem paths.
package categories

import (
	"os"
	"path/filepath"
)

// Category describes a logical group of files to back up.
type Category struct {
	// ID is the unique machine-readable identifier.
	ID string
	// Label is a short human-readable name.
	Label string
	// Description explains what the category contains.
	Description string
	// Paths lists the home-relative paths belonging to this category.
	// An empty Paths means the category uses AutoDetect logic.
	Paths []string
}

// All returns the full list of supported backup categories.
// The home_dirs category has no static Paths; callers should use
// DiscoverHomeDirs to populate its paths at runtime.
func All() []Category {
	return []Category{
		{
			ID:          "home_dirs",
			Label:       "Home Directories",
			Description: "All top-level non-hidden directories in home (except packrat and snap)",
			Paths:       nil, // populated at runtime via DiscoverHomeDirs
		},
		{
			ID:          "ssh",
			Label:       "SSH",
			Description: "SSH keys and configuration (~/.ssh)",
			Paths:       []string{".ssh"},
		},
		{
			ID:    "shell",
			Label: "Shell",
			Description: "Shell configuration files (.bashrc, .zshrc, .profile, etc.)",
			Paths: []string{
				".bashrc",
				".zshrc",
				".profile",
				".bash_profile",
				".bash_aliases",
				".zprofile",
				".zshenv",
			},
		},
		{
			ID:          "ai_agents",
			Label:       "AI Agents",
			Description: "AI coding agents and assistants (.claude, .codex, .cursor, .aider, .opencode, etc.)",
			Paths: []string{
				".claude", ".claude.json", ".claude.json.backup",
				".codex", ".agents",
				".opencode",
				".aider", ".aider.conf.yml",
				".continue",
				".cursor", ".config/cursor",
				".codeium",
				".supermaven",
				".tabnine",
				".config/github-copilot",
				".antigravity",
				".config/openai",
				".config/gemini",
			},
		},
		{
			ID:          "editors",
			Label:       "Editors",
			Description: "Editor configuration (.vscode, nvim, vim, helix, etc.)",
			Paths:       []string{".vscode", ".config/nvim", ".vim", ".vimrc", ".config/helix"},
		},
		{
			ID:          "cloud",
			Label:       "Cloud & DevOps",
			Description: "Cloud CLI credentials and configs (.aws, .kube, gcloud, docker, terraform, etc.)",
			Paths: []string{
				".aws",
				".kube",
				".config/gcloud",
				".docker",
				".config/docker",
				".terraform.d",
				".config/doctl",
				".config/fly",
				".config/heroku",
				".azure",
			},
		},
	}
}

// AutoDetect reports whether any of the category's paths exist under home.
// For the home_dirs category it checks whether any non-hidden non-excluded
// directory exists in home.
func (c Category) AutoDetect(home string) bool {
	if c.ID == "home_dirs" {
		dirs, _ := DiscoverHomeDirs(home)
		return len(dirs) > 0
	}
	for _, p := range c.Paths {
		if _, err := os.Lstat(filepath.Join(home, p)); err == nil {
			return true
		}
	}
	return false
}

// excludedHomeDirs lists top-level home directory names that should never be
// included in the home_dirs category.
var excludedHomeDirs = map[string]bool{
	"packrat": true,
	"snap":       true,
}

// DiscoverHomeDirs returns the names of all non-hidden, non-excluded
// subdirectories directly inside home.
func DiscoverHomeDirs(home string) ([]string, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 0 && name[0] == '.' {
			continue // skip hidden
		}
		if excludedHomeDirs[name] {
			continue
		}
		dirs = append(dirs, name)
	}
	return dirs, nil
}

// WithHomeDirs returns a copy of the categories slice with the home_dirs
// category's Paths field populated from DiscoverHomeDirs.
func WithHomeDirs(cats []Category, home string) []Category {
	result := make([]Category, len(cats))
	copy(result, cats)
	for i, c := range result {
		if c.ID == "home_dirs" {
			dirs, _ := DiscoverHomeDirs(home)
			paths := make([]string, len(dirs))
			copy(paths, dirs)
			result[i].Paths = paths
		}
	}
	return result
}

// PruneRules returns the set of directory names and home-relative paths that
// should always be excluded from the archive.
type PruneRules struct {
	// DirNames is a set of directory names that are pruned wherever they appear.
	DirNames map[string]bool
	// RelPaths is a set of home-relative paths that are pruned.
	RelPaths map[string]bool
}

// DefaultPruneRules returns the built-in prune rules.
func DefaultPruneRules() PruneRules {
	return PruneRules{
		DirNames: map[string]bool{
			"node_modules": true,
			".cache":       true,
		},
		RelPaths: map[string]bool{
			".npm":                       true,
			".bun/install/cache":         true,
			".claude/cache":              true,
			".claude/image-cache":        true,
			".claude/plugins/cache":      true,
			".claude/debug":              true,
			".codex/log":                 true,
			".codex/tmp":                 true,
			".codex/shell_snapshots":     true,
			".config/gcloud/logs":        true,
			".config/gcloud/.install":    true,
			".kube/cache":                true,
			".aws/sso/cache":             true,
		},
	}
}
