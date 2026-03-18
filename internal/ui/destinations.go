package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// DestinationsModel shows the destinations list.
type DestinationsModel struct {
	destinations []service.Destination
	cursor       int
	loading      bool
	err          string
	spinner      spinner.Model
	width        int
	height       int
}

// NewDestinationsModel creates a new destinations list view.
func NewDestinationsModel() DestinationsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)
	return DestinationsModel{spinner: s}
}

// SetDestinations updates the destinations list.
func (m *DestinationsModel) SetDestinations(dests []service.Destination) {
	m.destinations = dests
	m.loading = false
	m.err = ""
	if m.cursor >= len(dests) && len(dests) > 0 {
		m.cursor = len(dests) - 1
	}
}

// SetError sets an error message.
func (m *DestinationsModel) SetError(err string) {
	m.loading = false
	m.err = err
}

// SetLoading sets the loading state.
func (m *DestinationsModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetSize updates the terminal size.
func (m *DestinationsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SelectedDestination returns the currently highlighted destination.
func (m DestinationsModel) SelectedDestination() *service.Destination {
	if len(m.destinations) == 0 || m.cursor >= len(m.destinations) {
		return nil
	}
	return &m.destinations[m.cursor]
}

// Init implements tea.Model.
func (m DestinationsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles destinations list input.
func (m DestinationsModel) Update(msg tea.Msg) (DestinationsModel, tea.Cmd) {
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
			if m.cursor < len(m.destinations)-1 {
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

// View renders the destinations list.
func (m DestinationsModel) View() string {
	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Destinations"),
			"",
			m.spinner.View()+" Loading destinations...",
		)
	}

	if m.err != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Destinations"),
			"",
			StyleError.Render("Error: "+m.err),
			"",
			StyleMuted.Render("Press r to refresh"),
		)
	}

	title := StyleTitle.Render(fmt.Sprintf("Destinations (%d)", len(m.destinations)))

	if len(m.destinations) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			StyleMuted.Render("No destinations found."),
			StyleHelp.Render("Press a to add a destination"),
		)
	}

	const (
		colID      = 6
		colName    = 24
		colType    = 14
		colVersion = 12
		colJobs    = 6
	)

	header := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s",
			colID, "ID",
			colName, "NAME",
			colType, "TYPE",
			colVersion, "VERSION",
			"JOBS",
		),
	)

	divider := StyleMuted.Render(strings.Repeat("─", colID+colName+colType+colVersion+colJobs+10))

	var rows []string
	rows = append(rows, header, divider)

	for i, d := range m.destinations {
		name := d.Name
		if len(name) > colName {
			name = name[:colName-1] + "…"
		}
		row := fmt.Sprintf("%-*d  %-*s  %-*s  %-*s  %d",
			colID, d.ID,
			colName, name,
			colType, d.Type,
			colVersion, d.Version,
			len(d.Jobs),
		)
		if i == m.cursor {
			rows = append(rows, StyleSelected.Render("> "+row))
		} else {
			rows = append(rows, StyleNormal.Render("  "+row))
		}
	}

	list := strings.Join(rows, "\n")
	help := StyleHelp.Render("a:add  e:edit  d:delete  t:test  r:refresh  enter:detail")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", help)
}
