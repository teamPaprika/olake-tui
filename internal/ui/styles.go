// Package ui provides Bubble Tea view components for the OLake TUI.
package ui

import "github.com/charmbracelet/lipgloss"

// OLake brand colors.
const (
	ColorCyan     = lipgloss.Color("#00BCD4") // OLake primary cyan
	ColorDarkCyan = lipgloss.Color("#0097A7")
	ColorBg       = lipgloss.Color("#0F1117") // near-black background
	ColorSurface  = lipgloss.Color("#1A1D27") // card/panel background
	ColorBorder   = lipgloss.Color("#2D3148") // subtle border
	ColorText     = lipgloss.Color("#E0E0E0") // primary text
	ColorMuted    = lipgloss.Color("#6B7A99") // secondary/muted text
	ColorSuccess  = lipgloss.Color("#4CAF50") // green
	ColorWarning  = lipgloss.Color("#FFC107") // amber
	ColorError    = lipgloss.Color("#F44336") // red
	ColorRunning  = lipgloss.Color("#2196F3") // blue
	ColorAccent   = ColorCyan
)

// Shared base styles.
var (
	// Base text styles
	StyleNormal = lipgloss.NewStyle().Foreground(ColorText)
	StyleMuted  = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleBold   = lipgloss.NewStyle().Foreground(ColorText).Bold(true)

	// Status styles
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleRunning = lipgloss.NewStyle().Foreground(ColorRunning)

	// Title style (cyan accent)
	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

	// Tab styles
	StyleTabActive = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true).
			Underline(true).
			Padding(0, 2)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	// Panel / card borders
	StylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorCyan).
				Padding(0, 1)

	// Status bar at the bottom
	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)

	// Help key hints
	StyleKey  = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	StyleHelp = lipgloss.NewStyle().Foreground(ColorMuted)

	// Toast / notification
	StyleToastSuccess = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0F1117")).
				Background(ColorSuccess).
				Padding(0, 1).
				Bold(true)

	StyleToastError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorError).
			Padding(0, 1).
			Bold(true)

	// Logo / header
	StyleLogo = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true).
			Padding(0, 1)

	// Selected row highlight
	StyleSelected = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)
)

// StatusColor returns the appropriate style for a job/task status string.
func StatusColor(status string) lipgloss.Style {
	switch status {
	case "WORKFLOW_EXECUTION_STATUS_COMPLETED", "completed", "success":
		return StyleSuccess
	case "WORKFLOW_EXECUTION_STATUS_FAILED", "failed", "error":
		return StyleError
	case "WORKFLOW_EXECUTION_STATUS_RUNNING", "running":
		return StyleRunning
	case "WORKFLOW_EXECUTION_STATUS_CANCELED", "cancelled", "canceled":
		return StyleWarning
	case "inactive", "paused":
		return StyleMuted
	default:
		return StyleNormal
	}
}

// JobStatusIcon returns a single-char icon for the status.
func JobStatusIcon(status string) string {
	switch status {
	case "WORKFLOW_EXECUTION_STATUS_COMPLETED", "completed", "success":
		return "✓"
	case "WORKFLOW_EXECUTION_STATUS_FAILED", "failed", "error":
		return "✗"
	case "WORKFLOW_EXECUTION_STATUS_RUNNING", "running":
		return "⟳"
	case "WORKFLOW_EXECUTION_STATUS_CANCELED", "cancelled", "canceled":
		return "⊘"
	default:
		return "·"
	}
}

// ActiveIcon returns a toggle icon.
func ActiveIcon(active bool) string {
	if active {
		return StyleSuccess.Render("●")
	}
	return StyleMuted.Render("○")
}
