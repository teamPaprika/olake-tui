package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LoginMsg is sent when the user submits the login form.
type LoginMsg struct {
	Username string
	Password string
}

// LoginModel is the login screen.
type LoginModel struct {
	usernameInput textinput.Model
	passwordInput textinput.Model
	focused       int // 0 = username, 1 = password
	err           string
	width         int
	height        int
}

// NewLoginModel creates a new login screen.
func NewLoginModel() LoginModel {
	u := textinput.New()
	u.Placeholder = "username"
	u.CharLimit = 100
	u.Width = 30
	u.Focus()

	p := textinput.New()
	p.Placeholder = "password"
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '•'
	p.CharLimit = 100
	p.Width = 30

	return LoginModel{
		usernameInput: u,
		passwordInput: p,
		focused:       0,
	}
}

// SetError sets an error message on the login screen.
func (m *LoginModel) SetError(err string) {
	m.err = err
}

// SetSize updates the terminal size.
func (m *LoginModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Init is the Bubble Tea Init for the login model (no-op).
func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles login screen input.
func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.err = "" // clear error on new keypress
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			m.focused = (m.focused + 1) % 2
			if m.focused == 0 {
				m.usernameInput.Focus()
				m.passwordInput.Blur()
			} else {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
			}
			return m, textinput.Blink
		case tea.KeyShiftTab, tea.KeyUp:
			m.focused = (m.focused + 1) % 2
			if m.focused == 0 {
				m.usernameInput.Focus()
				m.passwordInput.Blur()
			} else {
				m.passwordInput.Focus()
				m.usernameInput.Blur()
			}
			return m, textinput.Blink
		case tea.KeyEnter:
			if m.focused == 0 {
				// Advance to password
				m.focused = 1
				m.passwordInput.Focus()
				m.usernameInput.Blur()
				return m, textinput.Blink
			}
			// Submit
			return m, func() tea.Msg {
				return LoginMsg{
					Username: strings.TrimSpace(m.usernameInput.Value()),
					Password: m.passwordInput.Value(),
				}
			}
		}
	}

	var cmd tea.Cmd
	if m.focused == 0 {
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	} else {
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the login screen.
func (m LoginModel) View() string {
	logo := StyleLogo.Render("⬡ OLake")
	subtitle := StyleMuted.Render("Data Pipeline Management")

	header := lipgloss.JoinVertical(lipgloss.Center, logo, subtitle)

	labelStyle := lipgloss.NewStyle().Foreground(ColorText).Width(10)
	inputStyle := lipgloss.NewStyle().Foreground(ColorCyan)

	uLabel := labelStyle.Render("Username")
	pLabel := labelStyle.Render("Password")

	uField := uLabel + "  " + inputStyle.Render(m.usernameInput.View())
	pField := pLabel + "  " + inputStyle.Render(m.passwordInput.View())

	fields := lipgloss.JoinVertical(lipgloss.Left,
		"",
		uField,
		"",
		pField,
		"",
	)

	hint := StyleMuted.Render("tab: next field • enter: submit")

	var errMsg string
	if m.err != "" {
		errMsg = StyleToastError.Render(" ✗ "+m.err) + "\n"
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		fields,
		errMsg,
		hint,
	)

	box := StylePanel.
		Width(50).
		Align(lipgloss.Center).
		Render(content)

	// Center in terminal
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}
