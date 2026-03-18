package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StreamEntry is a simplified stream entry for display.
type StreamEntry struct {
	Namespace string
	Name      string
	SyncMode  string
	Selected  bool
}

// StreamsModel shows stream selection (placeholder for v1).
type StreamsModel struct {
	streams []StreamEntry
	cursor  int
	width   int
	height  int
}

// NewStreamsModel creates a new streams view.
func NewStreamsModel() StreamsModel {
	return StreamsModel{}
}

// SetStreams populates the streams list.
func (m *StreamsModel) SetStreams(streams []StreamEntry) {
	m.streams = streams
}

// SetSize updates terminal dimensions.
func (m *StreamsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// ToggleSelected toggles the currently highlighted stream.
func (m *StreamsModel) ToggleSelected() {
	if m.cursor < len(m.streams) {
		m.streams[m.cursor].Selected = !m.streams[m.cursor].Selected
	}
}

// MoveUp moves the cursor up.
func (m *StreamsModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the cursor down.
func (m *StreamsModel) MoveDown() {
	if m.cursor < len(m.streams)-1 {
		m.cursor++
	}
}

// View renders the streams selection screen.
func (m StreamsModel) View() string {
	title := StyleTitle.Render("Stream Selection")

	if len(m.streams) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			StyleMuted.Render("No streams available."),
			StyleMuted.Render("Run discover first (requires BFF server + Temporal)."),
		)
	}

	header := StyleMuted.Render(fmt.Sprintf("  %-4s  %-20s  %-24s  %s",
		"SEL", "NAMESPACE", "STREAM", "SYNC MODE"))
	divider := StyleMuted.Render(strings.Repeat("─", 70))

	var rows []string
	rows = append(rows, header, divider)

	for i, s := range m.streams {
		sel := "[ ]"
		if s.Selected {
			sel = StyleSuccess.Render("[x]")
		}
		row := fmt.Sprintf("%-4s  %-20s  %-24s  %s", sel, s.Namespace, s.Name, s.SyncMode)
		if i == m.cursor {
			rows = append(rows, StyleSelected.Render("> "+row))
		} else {
			rows = append(rows, "  "+row)
		}
	}

	list := strings.Join(rows, "\n")
	help := StyleHelp.Render("↑↓:move  space:toggle  enter:confirm  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", help)
}
