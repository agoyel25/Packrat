package tui

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"packrat/internal/archive"
)

// InspectModel is a bubbletea model for the scrollable inspect table.
type InspectModel struct {
	table    table.Model
	viewport viewport.Model
	ready    bool
}

// Init satisfies tea.Model.
func (m InspectModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the inspect view.
func (m InspectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.ready = true
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the inspect table.
func (m InspectModel) View() string {
	var sb strings.Builder
	sb.WriteString(m.table.View())
	sb.WriteString("\n")
	sb.WriteString(Muted.Render("  ↑/↓ scroll · q quit"))
	sb.WriteString("\n")
	return sb.String()
}

// RunInspectTable renders a scrollable table of inspect entries.
// Columns: Path, Type, Size
// Navigation: arrow keys / j/k to scroll, q/Esc to quit
func RunInspectTable(entries []archive.InspectEntry) error {
	if len(entries) == 0 {
		fmt.Println(Muted.Render("  Archive is empty."))
		return nil
	}

	// Build rows.
	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		entryType := fileTypeLabel(e.Mode, e.IsDir)
		sizeStr := ""
		if !e.IsDir {
			sizeStr = formatInspectBytes(e.Size)
		}

		// Truncate path if needed.
		p := e.Path
		const maxPath = 52
		if len(p) > maxPath {
			p = "…" + p[len(p)-maxPath+1:]
		}

		rows = append(rows, table.Row{p, entryType, sizeStr})
	}

	// Define columns.
	cols := []table.Column{
		{Title: "Path", Width: 54},
		{Title: "Type", Width: 10},
		{Title: "Size", Width: 12},
	}

	// Build styled table.
	accentBg := lipgloss.Color("#7C9FE4")
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(accentBg).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C9FE4")).
		Background(lipgloss.Color("#1a1a2e")).
		Bold(true)

	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	tableStyles := table.Styles{
		Header:   headerStyle,
		Cell:     cellStyle,
		Selected: selectedStyle,
	}

	tableHeight := 20
	if len(rows) < tableHeight {
		tableHeight = len(rows) + 1
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithStyles(tableStyles),
	)

	m := InspectModel{
		table: t,
	}

	prog := tea.NewProgram(m)
	_, err := prog.Run()
	return err
}

// fileTypeLabel returns a display label for the file type.
func fileTypeLabel(mode fs.FileMode, isDir bool) string {
	if isDir {
		return Accent.Render("dir")
	}
	if mode&fs.ModeSymlink != 0 {
		return Muted.Italic(true).Render("symlink")
	}
	return "file"
}

// formatInspectBytes formats a byte count as a human-readable string.
func formatInspectBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return Muted.Render(fmt.Sprintf("%d B", b))
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return Muted.Render(fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp]))
}
