package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// JobLogsModel displays paginated logs for a job task.
type JobLogsModel struct {
	jobID        int
	taskID       string
	filePath     string
	logs         []service.LogEntry
	vp           viewport.Model
	loading      bool
	err          string
	spinner      spinner.Model
	width        int
	height       int
	olderCursor  int64
	newerCursor  int64
	hasMoreOlder bool
	hasMoreNewer bool
}

// NewJobLogsModel creates a new log viewer.
func NewJobLogsModel(jobID int, taskID, filePath string, width, height int) JobLogsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)

	vp := viewport.New(width, height-6)
	vp.Style = lipgloss.NewStyle()

	return JobLogsModel{
		jobID:    jobID,
		taskID:   taskID,
		filePath: filePath,
		spinner:  s,
		vp:       vp,
		width:    width,
		height:   height,
		loading:  true,
	}
}

// ApplyResponse populates the log viewer with a service response.
func (m *JobLogsModel) ApplyResponse(resp *service.TaskLogsResponse) {
	if resp == nil {
		return
	}
	m.logs = resp.Logs
	m.olderCursor = resp.OlderCursor
	m.newerCursor = resp.NewerCursor
	m.hasMoreOlder = resp.HasMoreOlder
	m.hasMoreNewer = resp.HasMoreNewer
	m.loading = false
	m.err = ""
	m.refreshContent()
}

// SetLoading toggles the loading spinner state.
func (m *JobLogsModel) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.err = ""
	}
}

// SetError sets an error state.
func (m *JobLogsModel) SetError(err string) {
	m.loading = false
	m.err = err
}

// SetSize updates terminal dimensions.
func (m *JobLogsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.vp.Width = w
	m.vp.Height = h - 6
}

// JobID returns the associated job identifier.
func (m JobLogsModel) JobID() int { return m.jobID }

// TaskID returns the associated task identifier.
func (m JobLogsModel) TaskID() string { return m.taskID }

// FilePath returns the worker log file path.
func (m JobLogsModel) FilePath() string { return m.filePath }

// OlderCursor reports the cursor for fetching older logs.
func (m JobLogsModel) OlderCursor() (int64, bool) { return m.olderCursor, m.hasMoreOlder }

// NewerCursor reports the cursor for fetching newer logs.
func (m JobLogsModel) NewerCursor() (int64, bool) { return m.newerCursor, m.hasMoreNewer }

// IsLoading reports whether a pagination request is pending.
func (m JobLogsModel) IsLoading() bool { return m.loading }

func (m *JobLogsModel) refreshContent() {
	var sb strings.Builder
	for _, l := range m.logs {
		levelStyle := StyleMuted
		switch strings.ToLower(l.Level) {
		case "error", "fatal":
			levelStyle = StyleError
		case "warn", "warning":
			levelStyle = StyleWarning
		case "info":
			levelStyle = StyleRunning
		case "debug":
			levelStyle = StyleMuted
		}
		line := fmt.Sprintf("%s %s  %s",
			StyleMuted.Render(l.Time),
			levelStyle.Render(fmt.Sprintf("%-5s", strings.ToUpper(l.Level))),
			StyleNormal.Render(l.Message),
		)
		sb.WriteString(line + "\n")
	}
	m.vp.SetContent(sb.String())
	m.vp.GotoBottom()
}

// Init implements tea.Model.
func (m JobLogsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles log viewer input.
func (m JobLogsModel) Update(msg tea.Msg) (JobLogsModel, tea.Cmd) {
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// View renders the log viewer.
func (m JobLogsModel) View() string {
	title := StyleTitle.Render(fmt.Sprintf("Logs — Job %d / Task %s", m.jobID, m.taskID))
	hint := StyleHelp.Render("↑↓/pgup/pgdn: scroll  •  p:older  •  n:newer  •  esc: back")

	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", m.spinner.View()+" Loading logs...", "", hint)
	}
	if m.err != "" {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", StyleError.Render("Error: "+m.err), "", hint)
	}

	count := StyleMuted.Render(fmt.Sprintf("%d log entries", len(m.logs)))
	return lipgloss.JoinVertical(lipgloss.Left, title, count, m.vp.View(), "", hint)
}
