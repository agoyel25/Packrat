package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const asciiArt = `█████▄ ▄████▄ ▄█████ ██ ▄█▀ █████▄  ▄████▄ ██████
██▄▄█▀ ██▄▄██ ██     ████   ██▄▄██▄ ██▄▄██   ██
██     ██  ██ ▀█████ ██ ▀█▄ ██   ██ ██  ██   ██  `

// bannerGradient defines the per-line colors for the ASCII art (pink → blue).
var bannerGradient = []string{
	"#FF6B9D", // line 1 — pink
	"#C77DFF", // line 2 — purple
	"#7C9FE4", // line 3 — blue
}

// Banner returns the fully rendered application banner including subtitle.
func Banner() string {
	lines := strings.Split(asciiArt, "\n")
	rendered := make([]string, len(lines))
	for i, line := range lines {
		colorIdx := i
		if colorIdx >= len(bannerGradient) {
			colorIdx = len(bannerGradient) - 1
		}
		rendered[i] = lipgloss.NewStyle().
			Foreground(lipgloss.Color(bannerGradient[colorIdx])).
			Render(line)
	}
	art := strings.Join(rendered, "\n")

	subtitle := Muted.Render("  packrat · WSL Home Directory Archiver")

	accentColor := lipgloss.Color("#7C9FE4")
	separator := lipgloss.NewStyle().Foreground(accentColor).Render("  " + strings.Repeat("─", 41))

	return fmt.Sprintf("%s\n\n%s\n%s", art, subtitle, separator)
}
