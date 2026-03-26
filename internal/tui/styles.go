// Package tui provides terminal UI components using Charmbracelet libraries.
package tui

import "github.com/charmbracelet/lipgloss"

// Package-level style variables used across all TUI components.
var (
	// Accent is the primary accent style (soft blue).
	Accent = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#4A6FA5", Dark: "#7C9FE4"})

	// Success is used for positive outcomes (soft green).
	Success = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#4A7C40", Dark: "#A8CC8C"})

	// Muted is for secondary / de-emphasised text.
	Muted = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#666666"})

	// Error is for error messages (soft red).
	Error = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#E25D5D"})

	// Title renders section headings in bold accent colour.
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.AdaptiveColor{Light: "#4A6FA5", Dark: "#7C9FE4"})

	// Bold renders text in bold weight.
	Bold = lipgloss.NewStyle().Bold(true)

	// Panel wraps content in a subtle rounded border.
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
		Padding(0, 1)
)

// CategoryColors maps category IDs to their display color styles.
var CategoryColors = map[string]lipgloss.Style{
	"home_dirs": lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE66D")),
	"ssh":       lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")),
	"ai_agents": lipgloss.NewStyle().Foreground(lipgloss.Color("#C77DFF")),
	"shell":     lipgloss.NewStyle().Foreground(lipgloss.Color("#4ECDC4")),
	"editors":   lipgloss.NewStyle().Foreground(lipgloss.Color("#7C9FE4")),
}

// CategoryIcon returns an icon for each category ID.
func CategoryIcon(id string) string {
	icons := map[string]string{
		"home_dirs": "⌂",
		"ssh":       "🔑",
		"ai_agents": "◈",
		"shell":     "$",
		"editors":   "✏",
	}
	if icon, ok := icons[id]; ok {
		return icon
	}
	return "•"
}
