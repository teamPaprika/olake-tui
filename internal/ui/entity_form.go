package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EntityKind distinguishes source forms from destination forms.
type EntityKind int

const (
	EntityKindSource EntityKind = iota
	EntityKindDest
)

// EntityFormMode distinguishes create from edit.
type EntityFormMode int

const (
	EntityFormCreate EntityFormMode = iota
	EntityFormEdit
)

// entityFormStep is the current wizard step.
type entityFormStep int

const (
	entityFormStepMeta       entityFormStep = iota // name + connector type
	entityFormStepConnection                       // connection fields
)

// EntityFormSubmitMsg is sent when the form is successfully submitted.
type EntityFormSubmitMsg struct {
	Kind      EntityKind
	Mode      EntityFormMode
	ID        int    // 0 for create
	Name      string
	Type      string
	Version   string
	ConfigJSON string // JSON-encoded connection config
}

// EntityFormCancelMsg is sent when the user cancels.
type EntityFormCancelMsg struct{}

// EntityFormModel is the multi-step create/edit form for sources and destinations.
type EntityFormModel struct {
	kind   EntityKind
	mode   EntityFormMode
	id     int // entity ID when editing
	width  int
	height int

	step    entityFormStep
	version string // stored version for edit mode

	// Step 1: meta fields
	nameInput      textinput.Model
	connectorIdx   int // index into connector labels/types slice
	connectorFocus bool // true = name input focused, false = connector selector focused
	metaFocusField int // 0 = name, 1 = connector selector

	// Step 2: connection fields
	connForm *FormModel

	err string
}

// NewEntityFormModel creates a new entity form for creating a source or destination.
func NewEntityFormModel(kind EntityKind, w, h int) EntityFormModel {
	ti := textinput.New()
	ti.Placeholder = "My Source"
	ti.CharLimit = 100
	ti.Width = 40
	ti.Focus()

	return EntityFormModel{
		kind:           kind,
		mode:           EntityFormCreate,
		width:          w,
		height:         h,
		step:           entityFormStepMeta,
		nameInput:      ti,
		connectorIdx:   0,
		metaFocusField: 0,
	}
}

// NewEntityFormModelEdit creates a pre-filled form for editing an existing entity.
func NewEntityFormModelEdit(kind EntityKind, id int, name, connType, version, configJSON string, w, h int) EntityFormModel {
	ti := textinput.New()
	ti.Placeholder = "My Source"
	ti.CharLimit = 100
	ti.Width = 40
	ti.SetValue(name)
	ti.Blur()

	// Find connector index
	labels, types := connectorLabelTypes(kind)
	idx := 0
	for i, t := range types {
		if strings.EqualFold(t, connType) {
			idx = i
			break
		}
	}
	_ = labels

	// Build connection form pre-filled
	prefill := ParseConfigJSON(configJSON)
	connFields := connectorFields(kind, types[idx], prefill)
	cf := NewFormModel(fmt.Sprintf("%s Connection", displayConnType(types[idx])), connFields)

	return EntityFormModel{
		kind:           kind,
		mode:           EntityFormEdit,
		id:             id,
		version:        version,
		width:          w,
		height:         h,
		step:           entityFormStepMeta,
		nameInput:      ti,
		connectorIdx:   idx,
		metaFocusField: 0,
		connForm:       &cf,
	}
}

// SetSize updates the terminal dimensions.
func (m *EntityFormModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Name returns the current value of the name input.
func (m EntityFormModel) Name() string {
	return strings.TrimSpace(m.nameInput.Value())
}

// SetTriggerSubmit signals the form to submit on next update cycle.
// This is a no-op placeholder — actual submission flows through the form's
// own submit mechanism. Kept for future use.
func (m *EntityFormModel) SetTriggerSubmit(_ bool) {}

// SubmitCmd attempts to trigger form submission and returns the resulting Cmd.
// If the form is on the connection step and has a valid connForm, it fires the
// submit. Otherwise returns nil.
func (m *EntityFormModel) SubmitCmd() tea.Cmd {
	if m.step == entityFormStepConnection && m.connForm != nil {
		if cmd := m.connForm.SubmitCmd(); cmd != nil {
			return cmd
		}
	}
	return nil
}

// Init implements tea.Model.
func (m EntityFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all key events for the form.
func (m EntityFormModel) Update(msg tea.Msg) (EntityFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case entityFormStepMeta:
			return m.updateMeta(msg)
		case entityFormStepConnection:
			return m.updateConnection(msg)
		}
	case FormSubmitMsg:
		// Received from sub-form (connection step)
		return m.handleConnFormSubmit(msg)
	case FormCancelMsg:
		return m, func() tea.Msg { return EntityFormCancelMsg{} }
	}

	// Delegate textinput blinking
	if m.step == entityFormStepMeta && m.metaFocusField == 0 {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
	if m.step == entityFormStepConnection && m.connForm != nil {
		cf, cmd := m.connForm.Update(msg)
		m.connForm = &cf
		return m, cmd
	}

	return m, nil
}

func (m EntityFormModel) updateMeta(msg tea.KeyMsg) (EntityFormModel, tea.Cmd) {
	m.err = ""
	_, types := connectorLabelTypes(m.kind)

	switch msg.Type {
	case tea.KeyTab, tea.KeyDown:
		if m.metaFocusField == 0 {
			m.nameInput.Blur()
			m.metaFocusField = 1
		} else {
			m.nameInput.Focus()
			m.metaFocusField = 0
		}
		return m, textinput.Blink

	case tea.KeyShiftTab, tea.KeyUp:
		if m.metaFocusField == 1 {
			m.nameInput.Focus()
			m.metaFocusField = 0
		} else {
			m.nameInput.Blur()
			m.metaFocusField = 1
		}
		return m, textinput.Blink

	case tea.KeyLeft:
		if m.metaFocusField == 1 {
			m.connectorIdx = (m.connectorIdx - 1 + len(types)) % len(types)
			m.connForm = nil // reset conn form on type change
		}

	case tea.KeyRight:
		if m.metaFocusField == 1 {
			m.connectorIdx = (m.connectorIdx + 1) % len(types)
			m.connForm = nil
		}

	case tea.KeyEnter:
		if m.metaFocusField == 0 && strings.TrimSpace(m.nameInput.Value()) == "" {
			m.err = "Name is required"
			return m, nil
		}
		// Advance to connection step
		return m.advanceToConnection()

	case tea.KeyEsc:
		return m, func() tea.Msg { return EntityFormCancelMsg{} }
	}

	// Delegate to name input when focused
	if m.metaFocusField == 0 {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m EntityFormModel) advanceToConnection() (EntityFormModel, tea.Cmd) {
	if strings.TrimSpace(m.nameInput.Value()) == "" {
		m.err = "Name is required"
		return m, nil
	}
	_, types := connectorLabelTypes(m.kind)
	connType := types[m.connectorIdx]

	if m.connForm == nil {
		fields := connectorFields(m.kind, connType, nil)
		cf := NewFormModel(fmt.Sprintf("%s Connection", displayConnType(connType)), fields)
		m.connForm = &cf
	}
	m.step = entityFormStepConnection
	return m, textinput.Blink
}

func (m EntityFormModel) updateConnection(msg tea.KeyMsg) (EntityFormModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		// Go back to meta step
		m.step = entityFormStepMeta
		m.nameInput.Focus()
		m.metaFocusField = 0
		return m, textinput.Blink
	}

	if m.connForm != nil {
		cf, cmd := m.connForm.Update(msg)
		m.connForm = &cf

		// Check if FormSubmitMsg was triggered (cmd returns FormSubmitMsg)
		// We handle it via the FormSubmitMsg case in Update
		return m, cmd
	}
	return m, nil
}

func (m EntityFormModel) handleConnFormSubmit(msg FormSubmitMsg) (EntityFormModel, tea.Cmd) {
	_, types := connectorLabelTypes(m.kind)
	connType := types[m.connectorIdx]
	name := strings.TrimSpace(m.nameInput.Value())
	configJSON := BuildConfigJSON(msg.Values)

	ver := m.version
	if ver == "" {
		ver = "latest"
	}
	submitMsg := EntityFormSubmitMsg{
		Kind:       m.kind,
		Mode:       m.mode,
		ID:         m.id,
		Name:       name,
		Type:       connType,
		Version:    ver,
		ConfigJSON: configJSON,
	}
	return m, func() tea.Msg { return submitMsg }
}

// View renders the entity form.
func (m EntityFormModel) View() string {
	labels, types := connectorLabelTypes(m.kind)
	kindLabel := "Source"
	if m.kind == EntityKindDest {
		kindLabel = "Destination"
	}

	modeLabel := "Create"
	if m.mode == EntityFormEdit {
		modeLabel = "Edit"
	}

	title := StyleTitle.Render(fmt.Sprintf("%s %s", modeLabel, kindLabel))

	var body string
	switch m.step {
	case entityFormStepMeta:
		body = m.viewMeta(labels, types)
	case entityFormStepConnection:
		if m.connForm != nil {
			body = m.connForm.View()
			body += "\n" + StyleMuted.Render("esc: back to info")
		}
	}

	var errMsg string
	if m.err != "" {
		errMsg = "\n" + StyleToastError.Render(" ✗ "+m.err)
	}

	stepIndicator := "Step 1/2: Info"
	if m.step == entityFormStepConnection {
		stepIndicator = "Step 2/2: Connection"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		StyleMuted.Render(stepIndicator),
		"",
		body,
		errMsg,
	)
}

func (m EntityFormModel) viewMeta(labels, types []string) string {
	focusStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

	// Name field
	nameLabel := "Name:"
	if m.metaFocusField == 0 {
		nameLabel = focusStyle.Render("Name:")
	}
	nameRow := nameLabel + "  " + m.nameInput.View()

	// Connector selector
	connLabel := "Connector:"
	if m.metaFocusField == 1 {
		connLabel = focusStyle.Render("Connector:")
	}

	connDisplay := ""
	if len(types) > 0 {
		prev := (m.connectorIdx - 1 + len(labels)) % len(labels)
		next := (m.connectorIdx + 1) % len(labels)
		connDisplay = fmt.Sprintf("← %s  [ %s ]  %s →",
			StyleMuted.Render(labels[prev]),
			StyleSelected.Render(labels[m.connectorIdx]),
			StyleMuted.Render(labels[next]),
		)
	}

	connRow := connLabel + "  " + connDisplay

	hint := StyleMuted.Render("tab/↑↓: move  •  ←→: change connector  •  enter: next  •  esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		nameRow,
		"",
		connRow,
		"",
		hint,
	)
}

// --- helpers ---

func connectorLabelTypes(kind EntityKind) (labels, types []string) {
	if kind == EntityKindSource {
		return SourceConnectorLabels, SourceConnectorTypes
	}
	return DestConnectorLabels, DestConnectorTypes
}

func connectorFields(kind EntityKind, connType string, prefill map[string]string) []FormField {
	if prefill == nil {
		prefill = map[string]string{}
	}
	if kind == EntityKindSource {
		return ConnectorFieldsForSource(connType, prefill)
	}
	return ConnectorFieldsForDest(connType, prefill)
}

func displayConnType(t string) string {
	switch strings.ToLower(t) {
	case ConnectorPostgres:
		return "PostgreSQL"
	case ConnectorMySQL:
		return "MySQL"
	case ConnectorMongoDB:
		return "MongoDB"
	case ConnectorOracle:
		return "Oracle"
	case ConnectorMSSQL:
		return "MSSQL"
	case ConnectorDB2:
		return "DB2"
	case ConnectorKafka:
		return "Kafka"
	case ConnectorS3:
		return "S3"
	case ConnectorIceberg:
		return "Apache Iceberg"
	case ConnectorParquet:
		return "Amazon S3 (Parquet)"
	default:
		if len(t) > 0 {
			return strings.ToUpper(t[:1]) + t[1:]
		}
		return t
	}
}
