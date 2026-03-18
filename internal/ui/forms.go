package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormField represents a single input field in a form.
type FormField struct {
	Label       string
	Placeholder string
	Value       string
	Secret      bool
	Required    bool
	input       textinput.Model
}

// FormSubmitMsg is sent when the form is submitted.
type FormSubmitMsg struct {
	Values map[string]string
}

// FormCancelMsg is sent when the form is cancelled.
type FormCancelMsg struct{}

// FormModel is a generic multi-field form component.
type FormModel struct {
	Title  string
	fields []FormField
	inputs []textinput.Model
	cursor int
	err    string
}

// NewFormModel creates a form with the given fields.
func NewFormModel(title string, fields []FormField) FormModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		t := textinput.New()
		t.Placeholder = f.Placeholder
		t.CharLimit = 500
		t.Width = 40
		if f.Secret {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		if f.Value != "" {
			t.SetValue(f.Value)
		}
		if i == 0 {
			t.Focus()
		}
		inputs[i] = t
	}
	return FormModel{
		Title:  title,
		fields: fields,
		inputs: inputs,
	}
}

// SetError sets a form-level error.
func (m *FormModel) SetError(err string) {
	m.err = err
}

// Init implements tea.Model.
func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles form input.
func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.err = ""
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor + 1) % len(m.inputs)
			m.inputs[m.cursor].Focus()
			return m, textinput.Blink
		case tea.KeyShiftTab, tea.KeyUp:
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.cursor].Focus()
			return m, textinput.Blink
		case tea.KeyEnter:
			if m.cursor < len(m.inputs)-1 {
				m.inputs[m.cursor].Blur()
				m.cursor++
				m.inputs[m.cursor].Focus()
				return m, textinput.Blink
			}
			// Last field — validate and submit inline so err is set on returned copy.
			values := make(map[string]string, len(m.fields))
			valid := true
			for i, f := range m.fields {
				val := strings.TrimSpace(m.inputs[i].Value())
				if f.Required && val == "" {
					m.err = f.Label + " is required"
					valid = false
					break
				}
				values[f.Label] = val
			}
			if !valid {
				return m, nil
			}
			captured := values
			return m, func() tea.Msg { return FormSubmitMsg{Values: captured} }
		case tea.KeyEsc:
			return m, func() tea.Msg { return FormCancelMsg{} }
		}
	}

	var cmd tea.Cmd
	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
	return m, cmd
}

// View renders the form.
func (m FormModel) View() string {
	title := StyleTitle.Render(m.Title)

	labelW := 0
	for _, f := range m.fields {
		if len(f.Label) > labelW {
			labelW = len(f.Label)
		}
	}

	labelStyle := lipgloss.NewStyle().Foreground(ColorText).Width(labelW + 2)
	var rows []string
	for i, f := range m.fields {
		label := labelStyle.Render(f.Label)
		input := m.inputs[i].View()
		if i == m.cursor {
			input = lipgloss.NewStyle().Foreground(ColorCyan).Render(input)
		}
		rows = append(rows, label+" "+input)
	}

	var errMsg string
	if m.err != "" {
		errMsg = "\n" + StyleToastError.Render(" ✗ "+m.err)
	}

	hint := StyleMuted.Render("tab: next  •  ↑↓: move  •  enter: submit  •  esc: cancel")

	content := strings.Join(rows, "\n")
	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		content,
		errMsg,
		"",
		hint,
	)

	return StylePanel.Width(60).Render(body)
}
