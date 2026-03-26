package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// FileProgressMsg is sent to the progress model when a file has been processed.
type FileProgressMsg struct {
	Count int
	Path  string
}

// DoneMsg is sent when the work goroutine finishes.
type DoneMsg struct{ Err error }

// tickMsg is sent every 100ms to update the elapsed timer.
type tickMsg time.Time

// BackupProgress describes a progress operation to display.
type BackupProgress struct {
	Title      string
	TotalFiles int // 0 = indeterminate
}

// ProgressModel is a Bubbletea model that shows live backup/restore progress.
type ProgressModel struct {
	title       string
	totalFiles  int
	filesCount  int
	currentPath string
	done        bool
	err         error
	spinner     spinner.Model
	progress    progress.Model
	startTime   time.Time
	elapsed     time.Duration
	tickCmd     tea.Cmd
}

// NewProgressModel creates a ProgressModel with the given BackupProgress config.
func NewProgressModel(bp BackupProgress) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = Accent

	p := progress.New(
		progress.WithGradient("#7C9FE4", "#A8CC8C"),
		progress.WithoutPercentage(),
	)
	p.Width = 50

	return ProgressModel{
		title:      bp.Title,
		totalFiles: bp.TotalFiles,
		spinner:    s,
		progress:   p,
		startTime:  time.Now(),
	}
}

// Init starts the spinner tick and elapsed ticker.
func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// Update handles incoming messages.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FileProgressMsg:
		m.filesCount = msg.Count
		m.currentPath = msg.Path
		return m, nil

	case DoneMsg:
		m.done = true
		m.err = msg.Err
		m.elapsed = time.Since(m.startTime)
		return m, tea.Quit

	case tickMsg:
		if m.done {
			return m, nil
		}
		m.elapsed = time.Since(m.startTime)

		var cmds []tea.Cmd
		// Schedule next tick.
		cmds = append(cmds, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

		// For indeterminate mode, slowly pulse the bar.
		if m.totalFiles == 0 {
			cmd := m.progress.IncrPercent(0.005)
			if m.progress.Percent() >= 1.0 {
				cmd = m.progress.SetPercent(0.0)
			}
			cmds = append(cmds, cmd)
		}

		// Forward progress frame messages.
		var progressCmd tea.Cmd
		_, progressCmd = m.progress.Update(msg)
		if progressCmd != nil {
			cmds = append(cmds, progressCmd)
		}

		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		var cmd tea.Cmd
		progressModel, progressCmd := m.progress.Update(msg)
		if pm, ok := progressModel.(progress.Model); ok {
			m.progress = pm
		}
		cmd = progressCmd
		return m, cmd

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the progress UI.
func (m ProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return Error.Render("  ✗ "+m.title+": "+m.err.Error()) + "\n"
		}
		// Done state: show summary.
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("  " + Success.Bold(true).Render("✓ Backup complete") + "\n")
		sb.WriteString("\n")
		statsLine := fmt.Sprintf("  %s · %s elapsed",
			Muted.Render(formatFilesProgress(m.filesCount)+" files"),
			Muted.Render(formatElapsed(m.elapsed)),
		)
		sb.WriteString(statsLine + "\n")
		return sb.String()
	}

	// Compute percentage string.
	var pct float64
	if m.totalFiles > 0 {
		pct = float64(m.filesCount) / float64(m.totalFiles)
		if pct > 1.0 {
			pct = 1.0
		}
	} else {
		pct = m.progress.Percent()
	}

	// Update the bar to current pct for determinate mode.
	barView := m.progress.ViewAs(pct)

	pctStr := fmt.Sprintf("%3.0f%%", pct*100)

	// Truncate path from left if too long.
	path := m.currentPath
	const maxPathLen = 55
	if len(path) > maxPathLen {
		path = "…" + path[len(path)-maxPathLen+1:]
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + Title.Render(m.title+"...") + "\n")
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%s]  %s\n", barView, Accent.Render(pctStr)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s  %s\n",
		m.spinner.View(),
		Muted.Render(path),
	))
	sb.WriteString(fmt.Sprintf("     %s · %s elapsed\n",
		Muted.Render(formatFilesProgress(m.filesCount)+" files"),
		Muted.Render(formatElapsed(m.elapsed)),
	))
	return sb.String()
}

// formatFilesProgress formats a file count with comma separators.
func formatFilesProgress(n int) string {
	s := fmt.Sprintf("%d", n)
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

// formatElapsed formats a duration as MM:SS.
func formatElapsed(d time.Duration) string {
	total := int(d.Seconds())
	if total < 0 {
		total = 0
	}
	minutes := total / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// RunProgress starts a Bubbletea program that shows live progress while work
// runs in a goroutine. The work function receives a progress callback it should
// call for each file processed.
func RunProgress(bp BackupProgress, work func(progress func(int, string)) error) error {
	model := NewProgressModel(bp)

	var p *tea.Program
	p = tea.NewProgram(model)

	// Run work in background goroutine; send messages via the program.
	go func() {
		err := work(func(count int, path string) {
			p.Send(FileProgressMsg{Count: count, Path: path})
		})
		p.Send(DoneMsg{Err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Propagate any error from the work function.
	if fm, ok := finalModel.(ProgressModel); ok && fm.err != nil {
		return fm.err
	}
	return nil
}
