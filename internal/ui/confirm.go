package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// ConfirmAction is the user's choice in a confirmation dialog.
type ConfirmAction int

const (
	ConfirmNone ConfirmAction = iota
	ConfirmYes
	ConfirmNo
)

// ConfirmModel is a yes/no confirmation dialog overlay.
type ConfirmModel struct {
	Title   string
	Message string
	focused int // 0 = yes, 1 = no
}

// NewConfirmModel creates a confirmation dialog.
func NewConfirmModel(title, message string) ConfirmModel {
	return ConfirmModel{Title: title, Message: message}
}

// HandleKey processes a key press and returns the chosen action.
func (c *ConfirmModel) HandleKey(key string) ConfirmAction {
	switch key {
	case "y", "Y":
		return ConfirmYes
	case "n", "N", "esc":
		return ConfirmNo
	case "left", "h":
		c.focused = 0
		return ConfirmNone
	case "right", "l":
		c.focused = 1
		return ConfirmNone
	case "enter":
		if c.focused == 0 {
			return ConfirmYes
		}
		return ConfirmNo
	}
	return ConfirmNone
}

// View renders the confirmation dialog.
func (c ConfirmModel) View(width, height int) string {
	title := StyleTitle.Render(c.Title)
	msg := StyleNormal.Render(c.Message)

	yesStyle := StyleSuccess
	noStyle := StyleMuted
	if c.focused == 1 {
		yesStyle = StyleMuted
		noStyle = StyleError
	}

	yes := yesStyle.Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Yes")
	no := noStyle.Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("No")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yes, "  ", no)

	hint := StyleMuted.Render("y/n: choose  •  ←→: move  •  enter: confirm")

	content := lipgloss.JoinVertical(lipgloss.Center, title, "", msg, "", buttons, "", hint)

	box := StylePanelFocused.
		Width(50).
		Align(lipgloss.Center).
		Render(content)

	if width > 0 && height > 0 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}
