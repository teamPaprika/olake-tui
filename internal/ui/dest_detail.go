package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Messages ─────────────────────────────────────────────────────────────────

// DestDetailBackMsg is sent when the user presses Esc.
type DestDetailBackMsg struct{}

// ─── Model ────────────────────────────────────────────────────────────────────

// DestDetailModel shows detailed info about a destination.
type DestDetailModel struct {
	dest   service.Destination
	jobs   []service.Job // jobs using this destination
	width  int
	height int
}

// NewDestDetailModel creates a destination detail view.
func NewDestDetailModel(dst service.Destination, allJobs []service.Job) DestDetailModel {
	var related []service.Job
	for _, j := range allJobs {
		if j.Destination.ID == dst.ID {
			related = append(related, j)
		}
	}
	return DestDetailModel{dest: dst, jobs: related}
}

// SetSize updates dimensions.
func (m *DestDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key events.
func (m DestDetailModel) Update(msg tea.Msg) (DestDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "esc" || msg.String() == "q" {
			return m, func() tea.Msg { return DestDetailBackMsg{} }
		}
	}
	return m, nil
}

// View renders the destination detail.
func (m DestDetailModel) View() string {
	d := m.dest
	title := StyleTitle.Render(fmt.Sprintf("Destination — %s", d.Name))

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(StyleMuted.Render(strings.Repeat("─", 60)) + "\n\n")

	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("ID:"), StyleBold.Render(fmt.Sprintf("%d", d.ID))))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Type:"), StyleNormal.Render(d.Type)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Version:"), StyleNormal.Render(d.Version)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Created:"), StyleMuted.Render(d.CreatedAt.Format("2006-01-02 15:04"))))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Updated:"), StyleMuted.Render(d.UpdatedAt.Format("2006-01-02 15:04"))))
	if d.CreatedBy != "" {
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Created by:"), StyleNormal.Render(d.CreatedBy)))
	}
	sb.WriteString(fmt.Sprintf("  %-16s %d\n", StyleMuted.Render("Job count:"), d.JobCount))

	// Config
	sb.WriteString("\n" + StyleBold.Render("  Connection Config") + "\n")
	sb.WriteString(renderConfig(d.Config))

	// Related jobs
	sb.WriteString("\n" + StyleBold.Render("  Related Jobs") + "\n")
	if len(m.jobs) == 0 {
		sb.WriteString(StyleMuted.Render("  (none)") + "\n")
	} else {
		for _, j := range m.jobs {
			icon := JobStatusIcon(j.LastRunState)
			sb.WriteString(fmt.Sprintf("  %s  %s ← %s  (%s)\n",
				StatusColor(j.LastRunState).Render(icon),
				StyleNormal.Render(j.Name),
				StyleMuted.Render(j.Source.Name),
				StyleMuted.Render(j.Frequency),
			))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("esc: back"))

	return sb.String()
}
