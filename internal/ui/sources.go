package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// SourcesModel shows the sources list.
type SourcesModel struct {
	sources  []service.Source
	cursor   int
	loading  bool
	err      string
	spinner  spinner.Model
	width    int
	height   int
}

// NewSourcesModel creates a new sources list view.
func NewSourcesModel() SourcesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)
	return SourcesModel{spinner: s}
}

// SetSources updates the sources list.
func (m *SourcesModel) SetSources(sources []service.Source) {
	m.sources = sources
	m.loading = false
	m.err = ""
	if m.cursor >= len(sources) && len(sources) > 0 {
		m.cursor = len(sources) - 1
	}
}

// SetError sets an error message.
func (m *SourcesModel) SetError(err string) {
	m.loading = false
	m.err = err
}

// SetLoading sets the loading state.
func (m *SourcesModel) SetLoading(loading bool) {
	m.loading = loading
}

// SetSize updates the terminal size.
func (m *SourcesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SelectedSource returns the currently highlighted source.
func (m SourcesModel) SelectedSource() *service.Source {
	if len(m.sources) == 0 || m.cursor >= len(m.sources) {
		return nil
	}
	return &m.sources[m.cursor]
}

// Init implements tea.Model.
func (m SourcesModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles sources list input.
func (m SourcesModel) Update(msg tea.Msg) (SourcesModel, tea.Cmd) {
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
			if m.cursor < len(m.sources)-1 {
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

// View renders the sources list.
func (m SourcesModel) View() string {
	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Sources"),
			"",
			m.spinner.View()+" Loading sources...",
		)
	}

	if m.err != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Sources"),
			"",
			StyleError.Render("Error: "+m.err),
			"",
			StyleMuted.Render("Press r to refresh"),
		)
	}

	title := StyleTitle.Render(fmt.Sprintf("Sources (%d)", len(m.sources)))

	if len(m.sources) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			StyleMuted.Render("No sources found."),
			StyleHelp.Render("Press a to add a source"),
		)
	}

	// Column widths
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

	for i, s := range m.sources {
		name := s.Name
		if len(name) > colName {
			name = name[:colName-1] + "…"
		}
		row := fmt.Sprintf("%-*d  %-*s  %-*s  %-*s  %d",
			colID, s.ID,
			colName, name,
			colType, s.Type,
			colVersion, s.Version,
			len(s.Jobs),
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
