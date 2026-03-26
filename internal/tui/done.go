package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DoneStats holds the statistics shown on the done screen.
type DoneStats struct {
	Files    int
	Size     int64
	Elapsed  time.Duration
	Archive  string
	Packages map[string]int // manager → count, may be nil
}

// doneTickMsg drives the spinner → checkmark transition.
type doneTickMsg struct{}

// DoneModel is an animated done screen that shows after backup completes.
type DoneModel struct {
	stats    DoneStats
	spinner  spinner.Model
	ticks    int
	finished bool // spinner → checkmark transition after 8 ticks
}

// newDoneModel creates a DoneModel with the given stats.
func newDoneModel(stats DoneStats) DoneModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = Success

	return DoneModel{
		stats:   stats,
		spinner: s,
	}
}

// Init starts the spinner and schedules the first done tick.
func (m DoneModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return doneTickMsg{}
		}),
	)
}

// Update handles messages for the done screen.
func (m DoneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case doneTickMsg:
		m.ticks++
		if m.ticks >= 8 {
			m.finished = true
			return m, tea.Quit
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
			return doneTickMsg{}
		})

	case tea.KeyMsg:
		m.finished = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the done screen.
func (m DoneModel) View() string {
	checkMark := "✓"
	if !m.finished {
		checkMark = m.spinner.View()
	}

	successBold := Success.Bold(true)
	header := fmt.Sprintf("  %s  %s",
		successBold.Render(checkMark),
		successBold.Render("Backup complete!"),
	)

	// Build panel content.
	accentColor := lipgloss.Color("#7C9FE4")
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE66D"))

	panelBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 3)

	labelStyle := Muted
	valueStyle := Bold

	var lines []string
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("  %-10s %s",
		labelStyle.Render("Archive"),
		valueStyle.Render(m.stats.Archive),
	))
	lines = append(lines, fmt.Sprintf("  %-10s %s",
		labelStyle.Render("Files"),
		valueStyle.Render(formatFilesProgress(m.stats.Files)),
	))
	lines = append(lines, fmt.Sprintf("  %-10s %s",
		labelStyle.Render("Size"),
		valueStyle.Render(formatDoneBytes(m.stats.Size)),
	))
	lines = append(lines, fmt.Sprintf("  %-10s %s",
		labelStyle.Render("Time"),
		valueStyle.Render(formatElapsed(m.stats.Elapsed)),
	))

	if len(m.stats.Packages) > 0 {
		// Build sorted package counts string.
		managers := make([]string, 0, len(m.stats.Packages))
		for mgr := range m.stats.Packages {
			managers = append(managers, mgr)
		}
		sort.Strings(managers)
		parts := make([]string, 0, len(managers))
		for _, mgr := range managers {
			parts = append(parts, fmt.Sprintf("%s(%d)", mgr, m.stats.Packages[mgr]))
		}
		lines = append(lines, fmt.Sprintf("  %-10s %s",
			labelStyle.Render("Packages"),
			valueStyle.Render(strings.Join(parts, "  ")),
		))
	}

	lines = append(lines, "")
	lines = append(lines, "  "+labelStyle.Render("Restore:"))
	lines = append(lines, "  "+yellowStyle.Render("packrat restore "+m.stats.Archive))
	lines = append(lines, "")

	panel := panelBorder.Render(strings.Join(lines, "\n"))

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(header + "\n")
	sb.WriteString("\n")
	sb.WriteString(panel + "\n")
	return sb.String()
}

// formatDoneBytes formats a byte count as a human-readable string.
func formatDoneBytes(b int64) string {
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

// ShowDoneScreen runs the animated done screen with the given stats.
func ShowDoneScreen(stats DoneStats) error {
	m := newDoneModel(stats)
	prog := tea.NewProgram(m)
	finalModel, err := prog.Run()
	if err != nil {
		return err
	}
	// Print the final view (with checkmark) after program exits.
	if dm, ok := finalModel.(DoneModel); ok {
		dm.finished = true
		fmt.Print(dm.View())
	}
	return nil
}
