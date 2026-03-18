package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Message types ────────────────────────────────────────────────────────────

// JobDetailBackMsg is sent when the user presses Esc to go back.
type JobDetailBackMsg struct{}

// JobDetailSyncMsg is sent when the user triggers a sync.
type JobDetailSyncMsg struct{ JobID int }

// JobDetailCancelMsg is sent when the user cancels a running task.
type JobDetailCancelMsg struct{ JobID int }

// JobDetailLogsMsg is sent when the user wants to view task logs.
type JobDetailLogsMsg struct {
	JobID    int
	TaskIdx  int
	FilePath string
}

// ─── Model ────────────────────────────────────────────────────────────────────

// JobDetailModel shows task history for a job.
type JobDetailModel struct {
	job     service.Job
	tasks   []service.JobTask
	cursor  int
	pageSize int
	pageStart int
	loading bool
	err     string
	spinner spinner.Model
	width   int
	height  int
}

// NewJobDetailModel creates a job detail model.
func NewJobDetailModel(job service.Job) JobDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)

	return JobDetailModel{
		job:      job,
		spinner:  s,
		loading:  true,
		pageSize: 15,
	}
}

// SetTasks populates the task history list.
func (m *JobDetailModel) SetTasks(tasks []service.JobTask) {
	m.tasks = tasks
	m.loading = false
	m.err = ""
	m.cursor = 0
	m.pageStart = 0
}

// SetError sets an error state.
func (m *JobDetailModel) SetError(err string) {
	m.loading = false
	m.err = err
}

// SetSize updates dimensions.
func (m *JobDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Adjust page size based on height
	if h > 20 {
		m.pageSize = h - 16 // header + footer room
		if m.pageSize < 5 {
			m.pageSize = 5
		}
	}
}

// Update handles key events for the job detail screen.
func (m JobDetailModel) Update(msg tea.Msg) (JobDetailModel, tea.Cmd) {
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return JobDetailBackMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.pageStart {
					m.pageStart = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
				if m.cursor >= m.pageStart+m.pageSize {
					m.pageStart = m.cursor - m.pageSize + 1
				}
			}

		case "enter", "l":
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				task := m.tasks[m.cursor]
				idx := m.cursor
				fp := task.FilePath
				return m, func() tea.Msg {
					return JobDetailLogsMsg{JobID: m.job.ID, TaskIdx: idx, FilePath: fp}
				}
			}

		case "s":
			return m, func() tea.Msg { return JobDetailSyncMsg{JobID: m.job.ID} }

		case "c":
			return m, func() tea.Msg { return JobDetailCancelMsg{JobID: m.job.ID} }

		case "pgup":
			if m.pageStart > 0 {
				m.pageStart -= m.pageSize
				if m.pageStart < 0 {
					m.pageStart = 0
				}
				m.cursor = m.pageStart
			}

		case "pgdown":
			if m.pageStart+m.pageSize < len(m.tasks) {
				m.pageStart += m.pageSize
				if m.pageStart >= len(m.tasks) {
					m.pageStart = len(m.tasks) - 1
				}
				m.cursor = m.pageStart
			}
		}
	}
	return m, nil
}

// View renders the job detail screen.
func (m JobDetailModel) View() string {
	// ── Header ────────────────────────────────────────────────────────────────
	title := StyleTitle.Render(fmt.Sprintf("Job Detail — %s", m.job.Name))

	src := m.job.Source.Name
	if src == "" {
		src = fmt.Sprintf("id:%d", m.job.Source.ID)
	}
	dst := m.job.Destination.Name
	if dst == "" {
		dst = fmt.Sprintf("id:%d", m.job.Destination.ID)
	}

	statusStyle := StatusColor(m.job.LastRunState)
	activeStr := "active"
	if !m.job.Activate {
		activeStr = "paused"
	}

	infoLine := fmt.Sprintf("  %s → %s   Status: %s   Schedule: %s   (%s)",
		StyleBold.Render(src),
		StyleBold.Render(dst),
		statusStyle.Render(m.job.LastRunState),
		StyleMuted.Render(cronDescription(m.job.Frequency)),
		StyleMuted.Render(activeStr),
	)

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(infoLine + "\n")
	sb.WriteString(StyleMuted.Render(strings.Repeat("─", 80)) + "\n\n")

	// ── Loading / Error ───────────────────────────────────────────────────────
	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading task history...\n")
		return sb.String()
	}

	if m.err != "" {
		sb.WriteString(StyleError.Render("Error: "+m.err) + "\n")
		sb.WriteString(StyleHelp.Render("esc: back") + "\n")
		return sb.String()
	}

	// ── Table header ─────────────────────────────────────────────────────────
	const (
		colIdx    = 5
		colStatus = 14
		colStart  = 22
		colDur    = 12
		colType   = 10
	)

	header := StyleMuted.Render(fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %s",
		colIdx, "#",
		colStatus, "STATUS",
		colStart, "STARTED",
		colDur, "DURATION",
		"TYPE",
	))
	divider := StyleMuted.Render(strings.Repeat("─", colIdx+colStatus+colStart+colDur+colType+14))

	sb.WriteString(header + "\n")
	sb.WriteString(divider + "\n")

	if len(m.tasks) == 0 {
		sb.WriteString(StyleMuted.Render("  No task history available.") + "\n")
	} else {
		// Paginated task rows
		end := m.pageStart + m.pageSize
		if end > len(m.tasks) {
			end = len(m.tasks)
		}
		for i := m.pageStart; i < end; i++ {
			task := m.tasks[i]

			status := task.Status
			icon := JobStatusIcon(status)

			started := task.StartTime
			if len(started) > 19 {
				started = started[:19]
			}

			dur := task.Runtime
			if len(dur) > colDur {
				dur = dur[:colDur]
			}

			row := fmt.Sprintf("%s %-*d  %-*s  %-*s  %-*s  %s",
				StatusColor(status).Render(icon),
				colIdx, i+1,
				colStatus, status,
				colStart, started,
				colDur, dur,
				task.JobType,
			)

			if i == m.cursor {
				sb.WriteString(StyleSelected.Render("> ") + row + "\n")
			} else {
				sb.WriteString("  " + row + "\n")
			}
		}

		// Pagination indicator
		if len(m.tasks) > m.pageSize {
			pPage := m.pageStart/m.pageSize + 1
			pTotal := (len(m.tasks) + m.pageSize - 1) / m.pageSize
			sb.WriteString("\n" + StyleMuted.Render(fmt.Sprintf("  Page %d/%d (%d tasks total)  pgup/pgdn: page", pPage, pTotal, len(m.tasks))) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("↑↓/j/k: navigate  •  enter/l: logs  •  s: sync  •  c: cancel  •  esc: back"))

	return sb.String()
}
