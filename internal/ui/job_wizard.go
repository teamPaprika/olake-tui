package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// WizardStep identifies which step of the wizard is active.
type WizardStep int

const (
	WizardStepConfig WizardStep = iota
	WizardStepSource
	WizardStepDest
	WizardStepStreams
)

// WizardMsg signals a completed wizard action back to the app.
type WizardMsg struct {
	Action WizardAction
	// For WizardActionDone: the created job id
	JobID int
	// For WizardActionDiscoverRequest
	SourceID int
}

// WizardAction enumerates what the wizard needs from its parent.
type WizardAction int

const (
	WizardActionCancel WizardAction = iota
	WizardActionDiscover              // parent should call service.DiscoverStreams then send WizardStreamsLoaded
	WizardActionDone                  // job created; JobID is set
)

// WizardStreamsLoaded is sent to the wizard after discover completes.
type WizardStreamsLoaded struct {
	Streams []service.StreamInfo
	Err     error
}

// WizardJobCreated is sent to the wizard after job creation.
type WizardJobCreated struct {
	JobID int
	Err   error
}

// ─── SelectList ───────────────────────────────────────────────────────────────

// selectList is a small list widget for selecting one item from a slice.
type selectList struct {
	items    []string // display labels
	cursor   int
	selected int // -1 = nothing selected
	focused  bool
}

func newSelectList(items []string) selectList {
	return selectList{items: items, selected: -1}
}

func (sl *selectList) handleKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if sl.cursor > 0 {
			sl.cursor--
		}
	case "down", "j":
		if sl.cursor < len(sl.items)-1 {
			sl.cursor++
		}
	case "enter", " ":
		sl.selected = sl.cursor
	}
}

func (sl selectList) view(height int) string {
	if len(sl.items) == 0 {
		return StyleMuted.Render("  (none available)")
	}
	vm := height
	if vm <= 0 {
		vm = 8
	}

	scrollTop := 0
	if sl.cursor >= scrollTop+vm {
		scrollTop = sl.cursor - vm + 1
	}

	var lines []string
	for i, item := range sl.items {
		if i < scrollTop || len(lines) >= vm {
			continue
		}
		sel := " "
		if i == sl.selected {
			sel = StyleSuccess.Render("●")
		}
		line := fmt.Sprintf("%s  %s", sel, item)
		if i == sl.cursor && sl.focused {
			lines = append(lines, StyleSelected.Render("> "+line))
		} else {
			lines = append(lines, "  "+line)
		}
	}
	return strings.Join(lines, "\n")
}

// ─── JobWizardModel ───────────────────────────────────────────────────────────

// JobWizardModel implements the 4-step job creation wizard.
type JobWizardModel struct {
	step WizardStep

	// Step 1 – Config
	jobNameInput textinput.Model
	srcList      selectList
	dstList      selectList
	configFocus  int // 0=name, 1=srcList, 2=dstList, 3=next
	configErr    string

	// Step 2 – Source review
	sources []service.Source

	// Step 3 – Dest review
	dests []service.Destination

	// Step 4 – Streams
	streamsModel StreamsModel
	discovering  bool
	discoverErr  string
	discSpinner  spinner.Model

	// Cancel confirm
	showCancel bool

	// Layout
	width  int
	height int
}

// NewJobWizardModel creates a new wizard with the loaded sources and destinations.
func NewJobWizardModel(sources []service.Source, dests []service.Destination, width, height int) JobWizardModel {
	ni := textinput.New()
	ni.Placeholder = "my_sync_job"
	ni.Width = 32
	ni.Focus()

	srcLabels := make([]string, len(sources))
	for i, s := range sources {
		srcLabels[i] = fmt.Sprintf("%-20s  (%s)", s.Name, s.Type)
	}
	dstLabels := make([]string, len(dests))
	for i, d := range dests {
		dstLabels[i] = fmt.Sprintf("%-20s  (%s)", d.Name, d.Type)
	}

	sl := newSelectList(srcLabels)
	sl.focused = false

	dl := newSelectList(dstLabels)
	dl.focused = false

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorCyan)

	sm := NewStreamsModel()
	sm.SetSize(width, height-12)

	m := JobWizardModel{
		step:         WizardStepConfig,
		jobNameInput: ni,
		srcList:      sl,
		dstList:      dl,
		sources:      sources,
		dests:        dests,
		discSpinner:  sp,
		streamsModel: sm,
		width:        width,
		height:       height,
	}
	m.configFocus = 0
	m.updateConfigFocus()
	return m
}

// JobName returns the entered job name.
func (m *JobWizardModel) JobName() string {
	return strings.TrimSpace(m.jobNameInput.Value())
}

// SelectedSourceID returns the selected source ID, or 0 if none.
func (m *JobWizardModel) SelectedSourceID() int {
	src := m.selectedSource()
	if src == nil {
		return 0
	}
	return src.ID
}

// SelectedDestID returns the selected destination ID, or 0 if none.
func (m *JobWizardModel) SelectedDestID() int {
	dst := m.selectedDest()
	if dst == nil {
		return 0
	}
	return dst.ID
}

// SelectedStreamConfigs returns the user-chosen stream configurations.
func (m *JobWizardModel) SelectedStreamConfigs() []service.StreamConfig {
	return m.streamsModel.GetSelectedConfigs()
}

// SetSize updates the terminal dimensions.
func (m *JobWizardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.streamsModel.SetSize(w, h-12)
}

// updateConfigFocus routes focus to the correct widget in step 1.
func (m *JobWizardModel) updateConfigFocus() {
	m.jobNameInput.Blur()
	m.srcList.focused = false
	m.dstList.focused = false
	switch m.configFocus {
	case 0:
		m.jobNameInput.Focus()
	case 1:
		m.srcList.focused = true
	case 2:
		m.dstList.focused = true
	}
}

// selectedSource returns the chosen source (may be nil).
func (m *JobWizardModel) selectedSource() *service.Source {
	if m.srcList.selected < 0 || m.srcList.selected >= len(m.sources) {
		return nil
	}
	return &m.sources[m.srcList.selected]
}

// selectedDest returns the chosen destination (may be nil).
func (m *JobWizardModel) selectedDest() *service.Destination {
	if m.dstList.selected < 0 || m.dstList.selected >= len(m.dests) {
		return nil
	}
	return &m.dests[m.dstList.selected]
}

// Init implements tea.Model.
func (m JobWizardModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.discSpinner.Tick)
}

// Update handles all keyboard events for the wizard.
func (m JobWizardModel) Update(msg tea.Msg) (JobWizardModel, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.discSpinner, cmd = m.discSpinner.Update(msg)
		return m, cmd

	case WizardStreamsLoaded:
		m.discovering = false
		if msg.Err != nil {
			m.discoverErr = msg.Err.Error()
		} else {
			m.streamsModel.SetStreams(msg.Streams)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate textinput tick
	if m.step == WizardStepConfig && m.configFocus == 0 {
		var cmd tea.Cmd
		m.jobNameInput, cmd = m.jobNameInput.Update(msg)
		return m, cmd
	}

	// Delegate streams model when on streams step
	if m.step == WizardStepStreams && !m.discovering {
		var cmd tea.Cmd
		m.streamsModel, cmd = m.streamsModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m JobWizardModel) handleKey(msg tea.KeyMsg) (JobWizardModel, tea.Cmd) {
	// Cancel confirm dialog
	if m.showCancel {
		switch msg.String() {
		case "y", "Y", "enter":
			return m, func() tea.Msg { return WizardMsg{Action: WizardActionCancel} }
		case "n", "N", "esc":
			m.showCancel = false
		}
		return m, nil
	}

	// Global esc → cancel confirm
	if msg.String() == "esc" {
		// Allow streams popup to close first
		if m.step == WizardStepStreams && m.streamsModel.HasPopup() {
			var cmd tea.Cmd
			m.streamsModel, cmd = m.streamsModel.Update(msg)
			return m, cmd
		}
		m.showCancel = true
		return m, nil
	}

	switch m.step {

	// ── Step 1: Config ─────────────────────────────────────────────────────
	case WizardStepConfig:
		switch msg.String() {
		case "tab":
			m.configFocus = (m.configFocus + 1) % 4
			m.updateConfigFocus()
		case "shift+tab":
			m.configFocus = (m.configFocus + 3) % 4
			m.updateConfigFocus()
		case "enter":
			if m.configFocus == 3 {
				return m.advanceConfig()
			}
			// On src/dst list enter selects
			if m.configFocus == 1 {
				m.srcList.handleKey(msg)
			} else if m.configFocus == 2 {
				m.dstList.handleKey(msg)
			} else {
				// name input → move focus to src list
				m.configFocus = 1
				m.updateConfigFocus()
			}
		default:
			switch m.configFocus {
			case 0:
				var cmd tea.Cmd
				m.jobNameInput, cmd = m.jobNameInput.Update(msg)
				return m, cmd
			case 1:
				m.srcList.handleKey(msg)
			case 2:
				m.dstList.handleKey(msg)
			}
		}

	// ── Step 2: Source review ──────────────────────────────────────────────
	case WizardStepSource:
		switch msg.String() {
		case "n", "right":
			m.step = WizardStepDest
		case "p", "left", "b":
			m.step = WizardStepConfig
		}

	// ── Step 3: Destination review ────────────────────────────────────────
	case WizardStepDest:
		switch msg.String() {
		case "n", "right":
			return m.enterStreams()
		case "p", "left", "b":
			m.step = WizardStepSource
		}

	// ── Step 4: Streams ───────────────────────────────────────────────────
	case WizardStepStreams:
		switch msg.String() {
		case "p", "b":
			if !m.streamsModel.HasPopup() {
				m.step = WizardStepDest
				return m, nil
			}
		case "ctrl+enter":
			return m.createJob()
		default:
			// Delegate to streams model
			var cmd tea.Cmd
			m.streamsModel, cmd = m.streamsModel.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m JobWizardModel) advanceConfig() (JobWizardModel, tea.Cmd) {
	name := strings.TrimSpace(m.jobNameInput.Value())
	if name == "" {
		m.configErr = "Job name is required"
		return m, nil
	}
	if m.srcList.selected < 0 {
		m.configErr = "Please select a source"
		return m, nil
	}
	if m.dstList.selected < 0 {
		m.configErr = "Please select a destination"
		return m, nil
	}
	m.configErr = ""
	m.step = WizardStepSource
	return m, nil
}

func (m JobWizardModel) enterStreams() (JobWizardModel, tea.Cmd) {
	m.step = WizardStepStreams
	src := m.selectedSource()
	if src == nil {
		m.discoverErr = "No source selected"
		return m, nil
	}
	m.discovering = true
	m.discoverErr = ""
	m.streamsModel.SetLoading(true)

	srcID := src.ID
	return m, func() tea.Msg {
		return WizardMsg{Action: WizardActionDiscover, SourceID: srcID}
	}
}

func (m JobWizardModel) createJob() (JobWizardModel, tea.Cmd) {
	if m.streamsModel.SelectedCount() == 0 {
		m.discoverErr = "Select at least 1 stream before creating the job"
		return m, nil
	}
	// Return action to parent — parent will call service.CreateJob
	return m, func() tea.Msg {
		return WizardMsg{Action: WizardActionDone}
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

// View renders the full wizard.
func (m JobWizardModel) View() string {
	if m.showCancel {
		msg := "Cancel job creation? All progress will be lost."
		confirm := NewConfirmModel("Cancel Wizard", msg)
		return confirm.View(m.width, m.height)
	}

	indicator := m.renderStepIndicator()
	content := m.renderStep()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, indicator, "", content, "", footer)
}

// renderStepIndicator renders the "Config → Source → Destination → Streams" header.
func (m JobWizardModel) renderStepIndicator() string {
	steps := []struct {
		label string
		step  WizardStep
	}{
		{"1. Config", WizardStepConfig},
		{"2. Source", WizardStepSource},
		{"3. Destination", WizardStepDest},
		{"4. Streams", WizardStepStreams},
	}

	var parts []string
	for i, s := range steps {
		var label string
		if m.step == s.step {
			label = StyleTabActive.Render(s.label)
		} else if int(m.step) > i {
			label = StyleSuccess.Render(s.label)
		} else {
			label = StyleMuted.Render(s.label)
		}
		if i < len(steps)-1 {
			parts = append(parts, label, StyleMuted.Render(" → "))
		} else {
			parts = append(parts, label)
		}
	}

	bar := strings.Join(parts, "")
	title := StyleTitle.Render("New Job Wizard")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", bar)
}

// renderStep renders the current step's content.
func (m JobWizardModel) renderStep() string {
	switch m.step {
	case WizardStepConfig:
		return m.viewConfig()
	case WizardStepSource:
		return m.viewSourceReview()
	case WizardStepDest:
		return m.viewDestReview()
	case WizardStepStreams:
		return m.viewStreams()
	}
	return ""
}

func (m JobWizardModel) viewConfig() string {
	nameLabel := StyleBold.Render("Job Name:")
	nameField := m.jobNameInput.View()
	if m.configFocus == 0 {
		nameField = StylePanelFocused.Padding(0, 1).Render(nameField)
	}

	srcLabel := StyleBold.Render("Source:")
	srcBox := m.srcList.view(6)
	if m.configFocus == 1 {
		srcBox = StylePanelFocused.Padding(0, 1).Render(srcBox)
	} else {
		srcBox = StylePanel.Padding(0, 1).Render(srcBox)
	}

	dstLabel := StyleBold.Render("Destination:")
	dstBox := m.dstList.view(6)
	if m.configFocus == 2 {
		dstBox = StylePanelFocused.Padding(0, 1).Render(dstBox)
	} else {
		dstBox = StylePanel.Padding(0, 1).Render(dstBox)
	}

	nextStyle := StyleMuted.Bold(true)
	if m.configFocus == 3 {
		nextStyle = StyleSuccess.Bold(true)
	}
	nextBtn := nextStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Next →")

	var errLine string
	if m.configErr != "" {
		errLine = StyleError.Render("⚠ " + m.configErr)
	}

	hint := StyleHelp.Render("tab:next field  ↑↓/j/k:navigate list  enter:select  esc:cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		nameLabel, nameField, "",
		srcLabel, srcBox, "",
		dstLabel, dstBox, "",
		nextBtn,
		errLine, "",
		hint,
	)
}

func (m JobWizardModel) viewSourceReview() string {
	src := m.selectedSource()
	if src == nil {
		return StyleError.Render("No source selected")
	}

	title := StyleTitle.Render("Source: " + src.Name)
	typ := StyleMuted.Render("Type: ") + StyleBold.Render(src.Type)
	ver := StyleMuted.Render("Version: ") + src.Version
	created := StyleMuted.Render("Created by: ") + src.CreatedBy

	hint := StyleHelp.Render("n/→: next  p/←/b: back  esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		typ,
		ver,
		created, "",
		hint,
	)
}

func (m JobWizardModel) viewDestReview() string {
	dst := m.selectedDest()
	if dst == nil {
		return StyleError.Render("No destination selected")
	}

	title := StyleTitle.Render("Destination: " + dst.Name)
	typ := StyleMuted.Render("Type: ") + StyleBold.Render(dst.Type)
	ver := StyleMuted.Render("Version: ") + dst.Version
	created := StyleMuted.Render("Created by: ") + dst.CreatedBy

	hint := StyleHelp.Render("n/→: next (discover streams)  p/←/b: back  esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		typ,
		ver,
		created, "",
		hint,
	)
}

func (m JobWizardModel) viewStreams() string {
	if m.discovering {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Streams"),
			"",
			m.discSpinner.View()+" Discovering streams from source...",
		)
	}
	if m.discoverErr != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Streams"),
			"",
			StyleError.Render("⚠ "+m.discoverErr),
		)
	}

	streamsView := m.streamsModel.View()

	createStyle := StyleMuted.Bold(true)
	if m.streamsModel.SelectedCount() > 0 {
		createStyle = StyleSuccess.Bold(true)
	}
	createBtn := createStyle.
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		Render(fmt.Sprintf("Create Job  (%d streams selected)", m.streamsModel.SelectedCount()))

	footer := lipgloss.JoinHorizontal(lipgloss.Left,
		createBtn,
		"   ",
		StyleHelp.Render("ctrl+enter: create  p/b: back  esc: cancel"),
	)

	return lipgloss.JoinVertical(lipgloss.Left, streamsView, "", footer)
}

// renderFooter renders the wizard navigation footer.
func (m JobWizardModel) renderFooter() string {
	switch m.step {
	case WizardStepConfig:
		return StyleHelp.Render("Step 1/4: Configure job name, source, and destination")
	case WizardStepSource:
		return StyleHelp.Render("Step 2/4: Review source  •  n: next  p: back")
	case WizardStepDest:
		return StyleHelp.Render("Step 3/4: Review destination  •  n: next  p: back")
	case WizardStepStreams:
		return StyleHelp.Render("Step 4/4: Select streams  •  ctrl+enter: create job  p: back")
	}
	return ""
}
