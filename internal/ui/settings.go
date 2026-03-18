package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Message types ────────────────────────────────────────────────────────────

// SettingsSavedMsg is sent when settings are saved.
type SettingsSavedMsg struct {
	WebhookURL string
}

// SettingsCancelMsg is sent when the user cancels.
type SettingsCancelMsg struct{}

// ─── Focus ────────────────────────────────────────────────────────────────────

type settingsFocus int

const (
	sfFocusWebhook settingsFocus = iota
	sfFocusSave
	sfFocusCancel
	sfFocusMax
)

// ─── Model ────────────────────────────────────────────────────────────────────

// SettingsModel is the system settings screen.
type SettingsModel struct {
	webhookInput textinput.Model
	focus        settingsFocus
	version      string
	err          string
	width        int
	height       int
}

// NewSettingsModel creates a new settings model.
func NewSettingsModel(webhookURL, version string) SettingsModel {
	input := textinput.New()
	input.Placeholder = "https://hooks.slack.com/..."
	input.CharLimit = 500
	input.Width = 50
	input.SetValue(webhookURL)
	input.Focus()

	if version == "" {
		version = "unknown"
	}

	return SettingsModel{
		webhookInput: input,
		focus:        sfFocusWebhook,
		version:      version,
	}
}

// SetWebhookURL updates the webhook input value.
func (m *SettingsModel) SetWebhookURL(url string) {
	m.webhookInput.SetValue(url)
}

// SetError sets an error message.
func (m *SettingsModel) SetError(err string) {
	m.err = err
}

// SetSize updates terminal dimensions.
func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key events for the settings screen.
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return SettingsCancelMsg{} }

		case "tab", "down":
			m.focus = settingsFocus((int(m.focus) + 1) % int(sfFocusMax))
			m.applyFocus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.focus = settingsFocus((int(m.focus) - 1 + int(sfFocusMax)) % int(sfFocusMax))
			m.applyFocus()
			return m, textinput.Blink

		case "enter", " ":
			switch m.focus {
			case sfFocusSave:
				url := strings.TrimSpace(m.webhookInput.Value())
				return m, func() tea.Msg { return SettingsSavedMsg{WebhookURL: url} }
			case sfFocusCancel:
				return m, func() tea.Msg { return SettingsCancelMsg{} }
			case sfFocusWebhook:
				// Move to next on enter
				m.focus = sfFocusSave
				m.applyFocus()
				return m, textinput.Blink
			}
		}
	}

	// Delegate to text input when focused
	if m.focus == sfFocusWebhook {
		var cmd tea.Cmd
		m.webhookInput, cmd = m.webhookInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *SettingsModel) applyFocus() {
	if m.focus == sfFocusWebhook {
		m.webhookInput.Focus()
	} else {
		m.webhookInput.Blur()
	}
}

// View renders the system settings screen.
func (m SettingsModel) View() string {
	var sb strings.Builder

	title := StyleTitle.Render("⚙  System Settings")
	sb.WriteString(title + "\n\n")

	// ── Webhook section ───────────────────────────────────────────────────────
	sb.WriteString(StyleBold.Render("Webhook Notifications") + "\n")
	sb.WriteString(StyleMuted.Render("  Receive sync status alerts via webhook (Slack, Discord, etc.)") + "\n\n")

	webhookLabel := "Webhook URL"
	if m.focus == sfFocusWebhook {
		webhookLabel = StyleKey.Render(webhookLabel)
	} else {
		webhookLabel = StyleMuted.Render(webhookLabel)
	}
	sb.WriteString(fmt.Sprintf("  %-18s  %s\n\n", webhookLabel, m.webhookInput.View()))

	// ── Error ─────────────────────────────────────────────────────────────────
	if m.err != "" {
		sb.WriteString(StyleError.Render("  ✗ "+m.err) + "\n\n")
	}

	// ── Buttons ───────────────────────────────────────────────────────────────
	saveBtn := m.renderButton("Save", m.focus == sfFocusSave, true)
	cancelBtn := m.renderButton("Cancel", m.focus == sfFocusCancel, false)
	sb.WriteString("  " + saveBtn + "  " + cancelBtn + "\n\n")

	// ── Version info ──────────────────────────────────────────────────────────
	sb.WriteString(StyleMuted.Render(strings.Repeat("─", 60)) + "\n")
	sb.WriteString(StyleMuted.Render(fmt.Sprintf("  olake-tui v%s", m.version)) + "\n\n")

	// ── Help ──────────────────────────────────────────────────────────────────
	sb.WriteString(StyleHelp.Render("tab/↑↓: navigate  •  enter: activate  •  esc: cancel"))

	w := m.width - 4
	if w < 60 {
		w = 60
	}
	return StylePanel.Width(w).Render(sb.String())
}

func (m SettingsModel) renderButton(label string, focused bool, primary bool) string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	if primary {
		if focused {
			style = style.Foreground(ColorBg).Background(ColorCyan).BorderForeground(ColorCyan)
		} else {
			style = style.Foreground(ColorCyan).BorderForeground(ColorBorder)
		}
	} else {
		if focused {
			style = style.Foreground(ColorCyan).Bold(true).BorderForeground(ColorCyan)
		} else {
			style = style.Foreground(ColorMuted).BorderForeground(ColorBorder)
		}
	}
	return style.Render(label)
}
