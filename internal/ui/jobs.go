package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// JobsModel shows the jobs list.
type JobsModel struct {
	jobs    []service.Job
	cursor  int
	loading bool
	err     string
	spinner spinner.Model
	width   int
	height  int
}

// NewJobsModel creates a new jobs list view.
func NewJobsModel() JobsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)
	return JobsModel{spinner: s}
}

// SetJobs updates the jobs list.
func (m *JobsModel) SetJobs(jobs []service.Job) {
	m.jobs = jobs
	m.loading = false
	m.err = ""
	if m.cursor >= len(jobs) && len(jobs) > 0 {
		m.cursor = len(jobs) - 1
	}
}

// SetError sets an error message.
func (m *JobsModel) SetError(err string) {
	m.loading = false
	m.err = err
}

// SetLoading sets the loading state.
func (m *JobsModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetSize updates the terminal size.
func (m *JobsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SelectedJob returns the currently highlighted job.
func (m JobsModel) SelectedJob() *service.Job {
	if len(m.jobs) == 0 || m.cursor >= len(m.jobs) {
		return nil
	}
	return &m.jobs[m.cursor]
}

// Init implements tea.Model.
func (m JobsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles jobs list input.
func (m JobsModel) Update(msg tea.Msg) (JobsModel, tea.Cmd) {
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
			}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the jobs list.
func (m JobsModel) View() string {
	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Jobs"),
			"",
			m.spinner.View()+" Loading jobs...",
		)
	}

	if m.err != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Jobs"),
			"",
			StyleError.Render("Error: "+m.err),
			"",
			StyleMuted.Render("Press r to refresh"),
		)
	}

	title := StyleTitle.Render(fmt.Sprintf("Jobs (%d)", len(m.jobs)))

	if len(m.jobs) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			StyleMuted.Render("No jobs found."),
			StyleHelp.Render("Press n to create a new job"),
		)
	}

	const (
		colID     = 5
		colName   = 22
		colSrc    = 14
		colDst    = 14
		colStatus = 12
		colActive = 8
	)

	header := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
			colID, "ID",
			colName, "NAME",
			colSrc, "SOURCE",
			colDst, "DEST",
			colStatus, "LAST STATUS",
			"ACTIVE",
		),
	)

	divider := StyleMuted.Render(strings.Repeat("─", colID+colName+colSrc+colDst+colStatus+colActive+14))

	var rows []string
	rows = append(rows, header, divider)

	for i, j := range m.jobs {
		name := j.Name
		if len(name) > colName {
			name = name[:colName-1] + "…"
		}
		srcName := j.Source.Name
		if len(srcName) > colSrc {
			srcName = srcName[:colSrc-1] + "…"
		}
		dstName := j.Destination.Name
		if len(dstName) > colDst {
			dstName = dstName[:colDst-1] + "…"
		}

		// Shorten long Temporal status names
		status := j.LastRunState
		switch status {
		case "WORKFLOW_EXECUTION_STATUS_COMPLETED":
			status = "completed"
		case "WORKFLOW_EXECUTION_STATUS_FAILED":
			status = "failed"
		case "WORKFLOW_EXECUTION_STATUS_RUNNING":
			status = "running"
		case "WORKFLOW_EXECUTION_STATUS_CANCELED":
			status = "canceled"
		case "":
			status = "—"
		}

		icon := JobStatusIcon(j.LastRunState)
		activeIcon := ActiveIcon(j.Activate)

		row := fmt.Sprintf("%s %-*d  %-*s  %-*s  %-*s  %-*s  %s",
			StatusColor(j.LastRunState).Render(icon),
			colID, j.ID,
			colName, name,
			colSrc, srcName,
			colDst, dstName,
			colStatus, status,
			activeIcon,
		)

		if i == m.cursor {
			rows = append(rows, StyleSelected.Render("> ")+row)
		} else {
			rows = append(rows, "  "+row)
		}
	}

	list := strings.Join(rows, "\n")
	help := StyleHelp.Render("s:sync  c:cancel  l:logs  p:pause  d:delete  r:refresh")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", help)
}
