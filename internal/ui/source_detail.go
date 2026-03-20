package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Messages ─────────────────────────────────────────────────────────────────

// SourceDetailBackMsg is sent when the user presses Esc.
type SourceDetailBackMsg struct{}

// ─── Model ────────────────────────────────────────────────────────────────────

// SourceDetailModel shows detailed info about a source.
type SourceDetailModel struct {
	source service.Source
	jobs   []service.Job // jobs using this source
	width  int
	height int
}

// NewSourceDetailModel creates a source detail view.
func NewSourceDetailModel(src service.Source, allJobs []service.Job) SourceDetailModel {
	var related []service.Job
	for _, j := range allJobs {
		if j.Source.ID == src.ID {
			related = append(related, j)
		}
	}
	return SourceDetailModel{source: src, jobs: related}
}

// SetSize updates dimensions.
func (m *SourceDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key events.
func (m SourceDetailModel) Update(msg tea.Msg) (SourceDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "esc" || msg.String() == "q" {
			return m, func() tea.Msg { return SourceDetailBackMsg{} }
		}
	}
	return m, nil
}

// View renders the source detail.
func (m SourceDetailModel) View() string {
	s := m.source
	title := StyleTitle.Render(fmt.Sprintf("Source — %s", s.Name))

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(StyleMuted.Render(strings.Repeat("─", 60)) + "\n\n")

	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("ID:"), StyleBold.Render(fmt.Sprintf("%d", s.ID))))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Type:"), StyleNormal.Render(s.Type)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Version:"), StyleNormal.Render(s.Version)))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Created:"), StyleMuted.Render(s.CreatedAt.Format("2006-01-02 15:04"))))
	sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Updated:"), StyleMuted.Render(s.UpdatedAt.Format("2006-01-02 15:04"))))
	if s.CreatedBy != "" {
		sb.WriteString(fmt.Sprintf("  %-16s %s\n", StyleMuted.Render("Created by:"), StyleNormal.Render(s.CreatedBy)))
	}
	sb.WriteString(fmt.Sprintf("  %-16s %d\n", StyleMuted.Render("Job count:"), s.JobCount))

	// Config (formatted JSON)
	sb.WriteString("\n" + StyleBold.Render("  Connection Config") + "\n")
	sb.WriteString(renderConfig(s.Config))

	// Related jobs
	sb.WriteString("\n" + StyleBold.Render("  Related Jobs") + "\n")
	if len(m.jobs) == 0 {
		sb.WriteString(StyleMuted.Render("  (none)") + "\n")
	} else {
		for _, j := range m.jobs {
			icon := JobStatusIcon(j.LastRunState)
			sb.WriteString(fmt.Sprintf("  %s  %s → %s  (%s)\n",
				StatusColor(j.LastRunState).Render(icon),
				StyleNormal.Render(j.Name),
				StyleMuted.Render(j.Destination.Name),
				StyleMuted.Render(j.Frequency),
			))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("esc: back"))

	return sb.String()
}

// renderConfig formats a JSON config string for display.
func renderConfig(configJSON string) string {
	if configJSON == "" {
		return StyleMuted.Render("  (empty)") + "\n"
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &parsed); err != nil {
		return StyleMuted.Render("  " + configJSON) + "\n"
	}

	var sb strings.Builder
	for k, v := range parsed {
		val := fmt.Sprintf("%v", v)
		// Mask sensitive fields
		lower := strings.ToLower(k)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "key") {
			if len(val) > 4 {
				val = val[:2] + strings.Repeat("•", len(val)-4) + val[len(val)-2:]
			} else {
				val = "••••"
			}
		}
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", StyleMuted.Render(k+":"), StyleNormal.Render(val)))
	}
	return sb.String()
}
