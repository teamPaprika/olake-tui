package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Frequency options ────────────────────────────────────────────────────────

type FrequencyMode int

const (
	FreqManual FrequencyMode = iota
	FreqEveryMinutes
	FreqEveryHours
	FreqDaily
	FreqWeekly
	FreqCustom
)

var frequencyLabels = []string{
	"Manual",
	"Every X minutes",
	"Every X hours",
	"Daily",
	"Weekly",
	"Custom cron",
}

// toCron converts the form state to a cron expression string.
// Returns "" for manual (no schedule).
func toCron(mode FrequencyMode, number, hour, minute, customCron string) string {
	switch mode {
	case FreqManual:
		return ""
	case FreqEveryMinutes:
		n := number
		if n == "" {
			n = "30"
		}
		return fmt.Sprintf("*/%s * * * *", n)
	case FreqEveryHours:
		n := number
		if n == "" {
			n = "1"
		}
		return fmt.Sprintf("0 */%s * * *", n)
	case FreqDaily:
		h := hour
		mi := minute
		if h == "" {
			h = "0"
		}
		if mi == "" {
			mi = "0"
		}
		return fmt.Sprintf("%s %s * * *", mi, h)
	case FreqWeekly:
		h := hour
		mi := minute
		if h == "" {
			h = "0"
		}
		if mi == "" {
			mi = "0"
		}
		return fmt.Sprintf("%s %s * * 0", mi, h)
	case FreqCustom:
		return customCron
	}
	return ""
}

// cronDescription returns a short human-readable description of a cron expr.
func cronDescription(cron string) string {
	if cron == "" {
		return "No schedule (manual trigger only)"
	}
	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return "Invalid cron expression"
	}
	minute, hour, dom, month, dow := parts[0], parts[1], parts[2], parts[3], parts[4]
	// Handle common patterns
	switch {
	case strings.HasPrefix(minute, "*/") && hour == "*" && dom == "*" && month == "*" && dow == "*":
		return fmt.Sprintf("Every %s minutes", strings.TrimPrefix(minute, "*/"))
	case minute == "0" && strings.HasPrefix(hour, "*/") && dom == "*" && month == "*" && dow == "*":
		return fmt.Sprintf("Every %s hours", strings.TrimPrefix(hour, "*/"))
	case dom == "*" && month == "*" && dow == "*":
		return fmt.Sprintf("Daily at %s:%02s UTC", padZero(hour), padZero(minute))
	case dom == "*" && month == "*":
		days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		d := dow
		idx := 0
		fmt.Sscanf(d, "%d", &idx)
		dayName := d
		if idx >= 0 && idx < 7 {
			dayName = days[idx]
		}
		return fmt.Sprintf("Weekly on %s at %s:%02s UTC", dayName, padZero(hour), padZero(minute))
	default:
		return fmt.Sprintf("Cron: %s", cron)
	}
}

func padZero(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// ─── Message types ────────────────────────────────────────────────────────────

// JobSettingsSavedMsg is fired when settings are saved.
type JobSettingsSavedMsg struct {
	JobID     int
	Name      string
	Frequency string
}

// JobSettingsCancelMsg is fired when the user cancels.
type JobSettingsCancelMsg struct{}

// JobSettingsPauseMsg is fired to toggle pause state.
type JobSettingsPauseMsg struct {
	JobID    int
	Activate bool // new desired state
}

// JobSettingsClearDestMsg is fired to trigger clear destination.
type JobSettingsClearDestMsg struct{ JobID int }

// JobSettingsDeleteMsg is fired to delete the job.
type JobSettingsDeleteMsg struct{ JobID int }

// ─── Focus tracking ───────────────────────────────────────────────────────────

type jobSettingsFocus int

const (
	jsFocusName jobSettingsFocus = iota
	jsFocusFreq
	jsFocusNumber
	jsFocusHour
	jsFocusMinute
	jsFocusCustomCron
	jsFocusPause
	jsFocusClearDest
	jsFocusDelete
	jsFocusSave
	jsFocusCancel
	jsFocusMax
)

// ─── Model ────────────────────────────────────────────────────────────────────

// JobSettingsModel is the job settings editor screen.
type JobSettingsModel struct {
	job      service.Job
	focus    jobSettingsFocus
	freqMode FrequencyMode

	nameInput   textinput.Model
	numberInput textinput.Model
	hourInput   textinput.Model
	minuteInput textinput.Model
	cronInput   textinput.Model

	width  int
	height int
}

// NewJobSettingsModel creates a job settings model from an existing job.
func NewJobSettingsModel(job service.Job) JobSettingsModel {
	// Parse the stored cron string back into a mode + inputs
	freqMode, number, hour, minute, customCron := parseCron(job.Frequency)

	nameInput := textinput.New()
	nameInput.Placeholder = "Job name"
	nameInput.CharLimit = 100
	nameInput.Width = 40
	nameInput.SetValue(job.Name)
	nameInput.Focus()

	numberInput := textinput.New()
	numberInput.Placeholder = "30"
	numberInput.CharLimit = 4
	numberInput.Width = 10
	numberInput.SetValue(number)

	hourInput := textinput.New()
	hourInput.Placeholder = "0"
	hourInput.CharLimit = 2
	hourInput.Width = 6
	hourInput.SetValue(hour)

	minuteInput := textinput.New()
	minuteInput.Placeholder = "0"
	minuteInput.CharLimit = 2
	minuteInput.Width = 6
	minuteInput.SetValue(minute)

	cronInput := textinput.New()
	cronInput.Placeholder = "* * * * *"
	cronInput.CharLimit = 50
	cronInput.Width = 30
	cronInput.SetValue(customCron)

	return JobSettingsModel{
		job:         job,
		focus:       jsFocusName,
		freqMode:    freqMode,
		nameInput:   nameInput,
		numberInput: numberInput,
		hourInput:   hourInput,
		minuteInput: minuteInput,
		cronInput:   cronInput,
	}
}

// parseCron parses a cron string back into mode + inputs for the form.
func parseCron(cron string) (FrequencyMode, string, string, string, string) {
	if cron == "" {
		return FreqManual, "", "", "", ""
	}
	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return FreqCustom, "", "", "", cron
	}
	minute, hour, dom, month, dow := parts[0], parts[1], parts[2], parts[3], parts[4]

	switch {
	case strings.HasPrefix(minute, "*/") && hour == "*" && dom == "*" && month == "*" && dow == "*":
		return FreqEveryMinutes, strings.TrimPrefix(minute, "*/"), "", "", ""
	case minute == "0" && strings.HasPrefix(hour, "*/") && dom == "*" && month == "*" && dow == "*":
		return FreqEveryHours, strings.TrimPrefix(hour, "*/"), "", "", ""
	case dom == "*" && month == "*" && dow == "*":
		return FreqDaily, "", hour, minute, ""
	case dom == "*" && month == "*":
		return FreqWeekly, "", hour, minute, ""
	default:
		return FreqCustom, "", "", "", cron
	}
}

// SetSize updates dimensions.
func (m *JobSettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Job returns the job associated with this settings screen.
func (m JobSettingsModel) Job() service.Job {
	return m.job
}

// currentCron returns the cron string from current form state.
func (m JobSettingsModel) currentCron() string {
	return toCron(m.freqMode, m.numberInput.Value(), m.hourInput.Value(), m.minuteInput.Value(), m.cronInput.Value())
}

// focusableCount returns how many focusable items exist for current freqMode.
func (m JobSettingsModel) maxFocus() jobSettingsFocus {
	return jsFocusMax
}

// blurAll removes focus from all inputs.
func (m *JobSettingsModel) blurAll() {
	m.nameInput.Blur()
	m.numberInput.Blur()
	m.hourInput.Blur()
	m.minuteInput.Blur()
	m.cronInput.Blur()
}

// focusInput focuses the appropriate input for the given focus state.
func (m *JobSettingsModel) applyFocus() {
	m.blurAll()
	switch m.focus {
	case jsFocusName:
		m.nameInput.Focus()
	case jsFocusNumber:
		m.numberInput.Focus()
	case jsFocusHour:
		m.hourInput.Focus()
	case jsFocusMinute:
		m.minuteInput.Focus()
	case jsFocusCustomCron:
		m.cronInput.Focus()
	}
}

// Update handles key events for the job settings screen.
func (m JobSettingsModel) Update(msg tea.Msg) (JobSettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return JobSettingsCancelMsg{} }

		case "tab", "down":
			// advance focus, skipping irrelevant inputs based on freqMode
			m.focus = m.nextFocus(m.focus, 1)
			m.applyFocus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.focus = m.nextFocus(m.focus, -1)
			m.applyFocus()
			return m, textinput.Blink

		case "enter", " ":
			return m.handleActivate()

		case "left", "h":
			if m.focus == jsFocusFreq {
				if m.freqMode > 0 {
					m.freqMode--
				}
			}
			return m, nil

		case "right", "l":
			if m.focus == jsFocusFreq {
				if int(m.freqMode) < len(frequencyLabels)-1 {
					m.freqMode++
				}
			}
			return m, nil
		}
	}

	// Delegate to focused input
	var cmd tea.Cmd
	switch m.focus {
	case jsFocusName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case jsFocusNumber:
		m.numberInput, cmd = m.numberInput.Update(msg)
	case jsFocusHour:
		m.hourInput, cmd = m.hourInput.Update(msg)
	case jsFocusMinute:
		m.minuteInput, cmd = m.minuteInput.Update(msg)
	case jsFocusCustomCron:
		m.cronInput, cmd = m.cronInput.Update(msg)
	}
	return m, cmd
}

// nextFocus returns the next focus index, skipping irrelevant items.
func (m JobSettingsModel) nextFocus(cur jobSettingsFocus, dir int) jobSettingsFocus {
	for i := 0; i < int(jsFocusMax); i++ {
		next := jobSettingsFocus((int(cur) + dir + int(jsFocusMax)) % int(jsFocusMax))
		cur = next
		if m.isFocusable(next) {
			return next
		}
	}
	return cur
}

// isFocusable returns whether a focus item is relevant for the current freq mode.
func (m JobSettingsModel) isFocusable(f jobSettingsFocus) bool {
	switch f {
	case jsFocusNumber:
		return m.freqMode == FreqEveryMinutes || m.freqMode == FreqEveryHours
	case jsFocusHour, jsFocusMinute:
		return m.freqMode == FreqDaily || m.freqMode == FreqWeekly
	case jsFocusCustomCron:
		return m.freqMode == FreqCustom
	}
	return true
}

// handleActivate processes enter/space on buttons and freq selector.
func (m JobSettingsModel) handleActivate() (JobSettingsModel, tea.Cmd) {
	switch m.focus {
	case jsFocusSave:
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = m.job.Name
		}
		cron := m.currentCron()
		return m, func() tea.Msg {
			return JobSettingsSavedMsg{JobID: m.job.ID, Name: name, Frequency: cron}
		}
	case jsFocusCancel:
		return m, func() tea.Msg { return JobSettingsCancelMsg{} }
	case jsFocusPause:
		newState := !m.job.Activate
		return m, func() tea.Msg {
			return JobSettingsPauseMsg{JobID: m.job.ID, Activate: newState}
		}
	case jsFocusClearDest:
		return m, func() tea.Msg {
			return JobSettingsClearDestMsg{JobID: m.job.ID}
		}
	case jsFocusDelete:
		return m, func() tea.Msg {
			return JobSettingsDeleteMsg{JobID: m.job.ID}
		}
	case jsFocusFreq:
		// Enter cycles frequency mode forward
		m.freqMode = FrequencyMode((int(m.freqMode) + 1) % len(frequencyLabels))
	}
	return m, nil
}

// View renders the job settings screen.
func (m JobSettingsModel) View() string {
	var sb strings.Builder

	title := StyleTitle.Render(fmt.Sprintf("Job Settings — %s", m.job.Name))
	sb.WriteString(title + "\n\n")

	// ── Name ──────────────────────────────────────────────────────────────────
	sb.WriteString(m.renderField("Name", m.nameInput.View(), m.focus == jsFocusName))
	sb.WriteString("\n")

	// ── Schedule ─────────────────────────────────────────────────────────────
	sb.WriteString(StyleBold.Render("Schedule") + "\n")

	// Frequency selector
	freqBar := m.renderFreqSelector()
	focusedFreq := m.focus == jsFocusFreq
	freqLabel := "Frequency"
	if focusedFreq {
		freqLabel = StyleKey.Render(freqLabel)
	} else {
		freqLabel = StyleMuted.Render(freqLabel)
	}
	sb.WriteString(fmt.Sprintf("  %-18s %s\n", freqLabel, freqBar))

	// Conditional inputs
	switch m.freqMode {
	case FreqEveryMinutes:
		sb.WriteString(m.renderField("Every (minutes)", m.numberInput.View(), m.focus == jsFocusNumber))
		sb.WriteString("\n")
	case FreqEveryHours:
		sb.WriteString(m.renderField("Every (hours)", m.numberInput.View(), m.focus == jsFocusNumber))
		sb.WriteString("\n")
	case FreqDaily, FreqWeekly:
		hourView := m.hourInput.View()
		minView := m.minuteInput.View()
		focusHour := m.focus == jsFocusHour
		focusMin := m.focus == jsFocusMinute
		if focusHour {
			hourView = lipgloss.NewStyle().Foreground(ColorCyan).Render(hourView)
		}
		if focusMin {
			minView = lipgloss.NewStyle().Foreground(ColorCyan).Render(minView)
		}
		timeLabel := StyleMuted.Render("At (HH:MM)")
		sb.WriteString(fmt.Sprintf("  %-18s %s : %s\n", timeLabel, hourView, minView))
		sb.WriteString("\n")
	case FreqCustom:
		sb.WriteString(m.renderField("Cron expression", m.cronInput.View(), m.focus == jsFocusCustomCron))
		sb.WriteString("\n")
		sb.WriteString(StyleMuted.Render("  Format: minute hour day month weekday") + "\n")
	}

	// Cron preview
	cron := m.currentCron()
	preview := cronDescription(cron)
	sb.WriteString(StyleMuted.Render(fmt.Sprintf("  Preview: %s", preview)) + "\n\n")

	// ── Job Actions ───────────────────────────────────────────────────────────
	sb.WriteString(StyleBold.Render("Actions") + "\n")

	// Pause / Resume toggle
	pauseLabel := "Pause Job"
	if !m.job.Activate {
		pauseLabel = "Resume Job"
	}
	sb.WriteString("  " + m.renderButton(pauseLabel, m.focus == jsFocusPause, "warning") + "\n")
	sb.WriteString("  " + m.renderButton("Clear Destination", m.focus == jsFocusClearDest, "danger") + "\n")
	sb.WriteString("  " + m.renderButton("Delete Job", m.focus == jsFocusDelete, "danger") + "\n")
	sb.WriteString("\n")

	// ── Save / Cancel ─────────────────────────────────────────────────────────
	saveBtn := m.renderButton("Save", m.focus == jsFocusSave, "primary")
	cancelBtn := m.renderButton("Cancel", m.focus == jsFocusCancel, "secondary")
	sb.WriteString("  " + saveBtn + "  " + cancelBtn + "\n\n")

	// ── Help ──────────────────────────────────────────────────────────────────
	sb.WriteString(StyleHelp.Render("tab/↑↓: navigate  •  enter/space: select  •  ←→: cycle frequency  •  esc: back"))

	content := sb.String()
	return StylePanel.Width(m.effectiveWidth()).Render(content)
}

func (m JobSettingsModel) effectiveWidth() int {
	if m.width > 10 {
		return m.width - 4
	}
	return 76
}

func (m JobSettingsModel) renderField(label, input string, focused bool) string {
	var labelStr string
	if focused {
		labelStr = StyleKey.Render(fmt.Sprintf("  %-18s", label))
	} else {
		labelStr = StyleMuted.Render(fmt.Sprintf("  %-18s", label))
	}
	return labelStr + " " + input
}

func (m JobSettingsModel) renderFreqSelector() string {
	var parts []string
	for i, lbl := range frequencyLabels {
		if FrequencyMode(i) == m.freqMode {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(ColorCyan).Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorCyan).
				Padding(0, 1).
				Render(lbl))
		} else {
			parts = append(parts, StyleMuted.
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1).
				Render(lbl))
		}
	}
	return strings.Join(parts, " ")
}

func (m JobSettingsModel) renderButton(label string, focused bool, variant string) string {
	style := lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	switch variant {
	case "primary":
		if focused {
			style = style.Foreground(ColorBg).Background(ColorCyan).BorderForeground(ColorCyan)
		} else {
			style = style.Foreground(ColorCyan).BorderForeground(ColorBorder)
		}
	case "danger":
		if focused {
			style = style.Foreground(ColorBg).Background(ColorError).BorderForeground(ColorError)
		} else {
			style = style.Foreground(ColorError).BorderForeground(ColorBorder)
		}
	case "warning":
		if focused {
			style = style.Foreground(ColorBg).Background(ColorWarning).BorderForeground(ColorWarning)
		} else {
			style = style.Foreground(ColorWarning).BorderForeground(ColorBorder)
		}
	default:
		if focused {
			style = style.Foreground(ColorCyan).Bold(true).BorderForeground(ColorCyan)
		} else {
			style = style.Foreground(ColorMuted).BorderForeground(ColorBorder)
		}
	}
	return style.Render(label)
}
