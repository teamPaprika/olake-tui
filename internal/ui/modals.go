// Package ui — modal overlay system for the OLake TUI.
//
// All 17 modals from the UX spec (§18) are implemented here using a generic
// Modal interface + ModalOverlay renderer.  The app model stores a single
// *ModalState and calls ModalOverlay.View() on top of the current screen.
//
// Modal types:
//  1. TestConnectionModal          — spinner while testing
//  2. TestConnectionSuccessModal   — success checkmark, auto-dismiss
//  3. TestConnectionFailureModal   — error + expandable log lines
//  4. EntitySavedModal             — "saved successfully"
//  5. EntityCancelModal            — "discard changes?"
//  6. EntityEditModal              — warning: entity has active jobs
//  7. DeleteModal                  — delete source/destination (with job list)
//  8. DeleteJobModal               — delete a job
//  9. DestinationDatabaseModal     — S3 folder / Iceberg DB name editor
// 10. ClearDestinationModal        — "this will erase all data"
// 11. ClearDataModal               — secondary confirm for clear
// 12. SpecFailedModal              — spec fetch error
// 13. StreamDifferenceModal        — show added/removed streams
// 14. IngestionModeChangeModal     — confirm mode switch
// 15. ResetStreamsModal            — leave streams, lose progress
// 16. StreamEditDisabledModal      — editing disabled while clear is running
// 17. UpdatesModal                 — new OLake version notification
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Modal interface ──────────────────────────────────────────────────────────

// ModalAction is returned by a modal to signal intent to the parent.
type ModalAction int

const (
	ModalActionNone    ModalAction = iota
	ModalActionConfirm             // primary action (Yes / Confirm / OK)
	ModalActionCancel              // secondary action (No / Cancel / Close)
	ModalActionAlt                 // third action where applicable
	ModalActionDismiss             // auto-dismiss (no user action needed)
)

// Modal is the interface every modal type implements.
type Modal interface {
	// Update handles a key press and returns (updated modal, action).
	// The action tells the parent what to do next.
	Update(msg tea.KeyMsg) (Modal, ModalAction)

	// Tick is called for spinner / auto-dismiss ticks.
	Tick(t time.Time) (Modal, ModalAction)

	// View renders the modal content (NOT the overlay — ModalOverlay wraps it).
	View() string

	// NeedsSpinner reports whether this modal needs periodic Tick calls.
	NeedsSpinner() bool
}

// ─── ModalOverlay ─────────────────────────────────────────────────────────────

// ModalOverlay renders a Modal as a centred box over the full terminal.
func ModalOverlay(m Modal, width, height int) string {
	content := m.View()

	// Minimum overlay width
	overlayW := 60
	if width > 90 {
		overlayW = 70
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Padding(1, 2).
		Width(overlayW).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// ─── ModalTickMsg ─────────────────────────────────────────────────────────────

// ModalTickMsg is delivered by the auto-dismiss / spinner tick command.
type ModalTickMsg struct{ T time.Time }

// ModalSpinnerTick returns a tea.Cmd that fires a ModalTickMsg after d.
func ModalSpinnerTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return ModalTickMsg{T: t}
	})
}

// ─── 1. TestConnectionModal ───────────────────────────────────────────────────

// TestConnectionModal shows a spinner while connection testing is in progress.
// It cannot be dismissed by the user.
type TestConnectionModal struct {
	sp      spinner.Model
	message string
}

func NewTestConnectionModal() *TestConnectionModal {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorCyan)
	return &TestConnectionModal{sp: sp, message: "Testing your connection…"}
}

func (m *TestConnectionModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	// Not user-dismissable
	return m, ModalActionNone
}

func (m *TestConnectionModal) Tick(t time.Time) (Modal, ModalAction) {
	var cmd tea.Cmd
	m.sp, cmd = m.sp.Update(spinner.TickMsg{Time: t})
	_ = cmd
	return m, ModalActionNone
}

func (m *TestConnectionModal) View() string {
	title := StyleTitle.Render("Testing Connection")
	body := m.sp.View() + "  " + StyleNormal.Render(m.message)
	hint := StyleMuted.Render("Please wait…")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", body, "", hint)
}

func (m *TestConnectionModal) NeedsSpinner() bool { return true }

// ─── 2. TestConnectionSuccessModal ───────────────────────────────────────────

// TestConnectionSuccessModal is shown after a successful connection test.
// It auto-dismisses after 1 second.
type TestConnectionSuccessModal struct {
	elapsed time.Duration
}

func NewTestConnectionSuccessModal() *TestConnectionSuccessModal {
	return &TestConnectionSuccessModal{}
}

func (m *TestConnectionSuccessModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	return m, ModalActionDismiss
}

func (m *TestConnectionSuccessModal) Tick(t time.Time) (Modal, ModalAction) {
	m.elapsed += 250 * time.Millisecond
	if m.elapsed >= 1*time.Second {
		return m, ModalActionDismiss
	}
	return m, ModalActionNone
}

func (m *TestConnectionSuccessModal) View() string {
	icon := StyleSuccess.Bold(true).Render("✓")
	title := StyleTitle.Render("Connection Successful")
	body := StyleNormal.Render("Your connection has been verified.")
	hint := StyleMuted.Render("Continuing automatically…")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, "", hint)
}

func (m *TestConnectionSuccessModal) NeedsSpinner() bool { return true }

// ─── 3. TestConnectionFailureModal ───────────────────────────────────────────

// TestConnectionFailureModal is shown when connection test fails.
// "Back" = ModalActionCancel, "Edit" = ModalActionAlt (stay on form).
type TestConnectionFailureModal struct {
	errMsg   string
	logLines []string
	expanded bool
	focused  int // 0=Back, 1=Edit
}

func NewTestConnectionFailureModal(errMsg string, logLines []string) *TestConnectionFailureModal {
	return &TestConnectionFailureModal{errMsg: errMsg, logLines: logLines, focused: 1}
}

func (m *TestConnectionFailureModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc", "q":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "x", "e":
		m.expanded = !m.expanded
	case "enter":
		if m.focused == 0 {
			return m, ModalActionCancel
		}
		return m, ModalActionAlt // "Edit" — stay on form
	}
	return m, ModalActionNone
}

func (m *TestConnectionFailureModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *TestConnectionFailureModal) NeedsSpinner() bool                    { return false }

func (m *TestConnectionFailureModal) View() string {
	icon := StyleError.Bold(true).Render("✗")
	title := StyleTitle.Render("Connection Failed")

	errLine := StyleError.Render("Error: " + m.errMsg)

	var logsSection string
	if m.expanded && len(m.logLines) > 0 {
		var lines []string
		for _, l := range m.logLines {
			lines = append(lines, StyleMuted.Render("  "+l))
		}
		logsSection = "\n" + strings.Join(lines, "\n")
	}

	toggleHint := ""
	if len(m.logLines) > 0 {
		if m.expanded {
			toggleHint = StyleMuted.Render("x: hide logs")
		} else {
			toggleHint = StyleMuted.Render("x: show logs  (" + fmt.Sprintf("%d", len(m.logLines)) + " lines)")
		}
	}

	back := StyleMuted.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Back")
	edit := StyleMuted.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Edit")
	if m.focused == 0 {
		back = StyleWarning.Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Back")
	} else {
		edit = StyleSuccess.Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Edit")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, back, "  ", edit)
	hint := StyleMuted.Render("←→/tab: move  enter: select  esc: close")

	parts := []string{icon, title, "", errLine, logsSection}
	if toggleHint != "" {
		parts = append(parts, toggleHint)
	}
	parts = append(parts, "", buttons, "", hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ─── 4. EntitySavedModal ─────────────────────────────────────────────────────

// EntitySavedContext specifies what was saved (affects which buttons appear).
type EntitySavedContext int

const (
	EntitySavedSource      EntitySavedContext = iota
	EntitySavedDestination
	EntitySavedJob
)

// EntitySavedModal is shown after successfully creating a source, dest, or job.
// Confirm = primary nav ("Sources" / "Destinations" / "Jobs"),  Alt = "Create a job →"
type EntitySavedModal struct {
	ctx     EntitySavedContext
	name    string
	focused int // 0=primary, 1=alt (only when ctx is source/dest)
}

func NewEntitySavedModal(ctx EntitySavedContext, name string) *EntitySavedModal {
	return &EntitySavedModal{ctx: ctx, name: name}
}

func (m *EntitySavedModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	hasAlt := m.ctx != EntitySavedJob
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		if hasAlt {
			m.focused = 0
		}
	case "right", "l":
		if hasAlt {
			m.focused = 1
		}
	case "tab":
		if hasAlt {
			m.focused = (m.focused + 1) % 2
		}
	case "enter":
		if hasAlt && m.focused == 1 {
			return m, ModalActionAlt
		}
		return m, ModalActionConfirm
	}
	return m, ModalActionNone
}

func (m *EntitySavedModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *EntitySavedModal) NeedsSpinner() bool                    { return false }

func (m *EntitySavedModal) View() string {
	icon := StyleSuccess.Bold(true).Render("✓")
	title := StyleTitle.Render("Saved Successfully")

	var entityLabel string
	var primaryLabel string
	var altLabel string
	switch m.ctx {
	case EntitySavedSource:
		entityLabel = "Source"
		primaryLabel = "Go to Sources"
		altLabel = "Create a Job →"
	case EntitySavedDestination:
		entityLabel = "Destination"
		primaryLabel = "Go to Destinations"
		altLabel = "Create a Job →"
	case EntitySavedJob:
		entityLabel = "Job"
		primaryLabel = "Go to Jobs →"
	}

	badge := StyleSuccess.Render(entityLabel + " • Success")
	nameLabel := StyleBold.Render(m.name)
	body := nameLabel + "  " + badge

	primaryStyle := StyleSuccess.Bold(true)
	altStyle := StyleMuted

	if m.ctx != EntitySavedJob {
		if m.focused == 0 {
			primaryStyle = StyleSuccess.Bold(true)
			altStyle = StyleMuted
		} else {
			primaryStyle = StyleMuted
			altStyle = StyleTitle
		}
	}

	primaryBtn := primaryStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render(primaryLabel)

	var buttons string
	if m.ctx != EntitySavedJob {
		altBtn := altStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render(altLabel)
		buttons = lipgloss.JoinHorizontal(lipgloss.Center, primaryBtn, "  ", altBtn)
	} else {
		buttons = primaryBtn
	}

	hint := StyleMuted.Render("enter: select  ←→/tab: move")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, "", buttons, "", hint)
}

// ─── 5. EntityCancelModal ─────────────────────────────────────────────────────

// EntityCancelModalContext identifies what we're cancelling.
type EntityCancelContext int

const (
	EntityCancelSource  EntityCancelContext = iota
	EntityCancelDest
	EntityCancelJob
	EntityCancelJobEdit
)

// EntityCancelModal asks "Are you sure you want to cancel?"
// Confirm = stay ("Don't cancel"),  Cancel = leave.
type EntityCancelModal struct {
	ctx     EntityCancelContext
	focused int // 0=Don't cancel (stay), 1=Cancel (leave)
}

func NewEntityCancelModal(ctx EntityCancelContext) *EntityCancelModal {
	return &EntityCancelModal{ctx: ctx, focused: 0}
}

func (m *EntityCancelModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc", "n", "N":
		return m, ModalActionConfirm // "Don't cancel" = stay
	case "y", "Y":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm // Don't cancel
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *EntityCancelModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *EntityCancelModal) NeedsSpinner() bool                    { return false }

func (m *EntityCancelModal) View() string {
	var entityName string
	switch m.ctx {
	case EntityCancelSource:
		entityName = "source"
	case EntityCancelDest:
		entityName = "destination"
	case EntityCancelJob:
		entityName = "job"
	case EntityCancelJobEdit:
		entityName = "job edit"
	}
	title := StyleTitle.Render("Discard Changes?")
	body := StyleNormal.Render(fmt.Sprintf("Are you sure you want to cancel the %s?", entityName))
	sub := StyleMuted.Render("All unsaved changes will be lost.")

	stayStyle := StyleMuted
	leaveStyle := StyleError
	if m.focused == 0 {
		stayStyle = StyleSuccess.Bold(true)
		leaveStyle = StyleMuted
	} else {
		stayStyle = StyleMuted
		leaveStyle = StyleError.Bold(true)
	}

	stayBtn := stayStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Don't Cancel")
	leaveBtn := leaveStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, stayBtn, "  ", leaveBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: select  n: stay  y: leave")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", body, sub, "", buttons, "", hint)
}

// ─── 6. EntityEditModal ───────────────────────────────────────────────────────

// EntityEditModal warns that editing an entity may affect active jobs.
// Confirm = proceed,  Cancel = go back.
type EntityEditModal struct {
	entityName string
	jobNames   []string
	focused    int // 0=Confirm, 1=Cancel
}

func NewEntityEditModal(entityName string, jobNames []string) *EntityEditModal {
	return &EntityEditModal{entityName: entityName, jobNames: jobNames, focused: 1}
}

func (m *EntityEditModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *EntityEditModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *EntityEditModal) NeedsSpinner() bool                    { return false }

func (m *EntityEditModal) View() string {
	icon := StyleWarning.Bold(true).Render("⚠")
	title := StyleTitle.Render("Jobs May Be Affected")
	body := StyleNormal.Render(fmt.Sprintf("Editing '%s' may affect the following jobs:", m.entityName))
	warn := StyleWarning.Render("Ongoing jobs may fail if the entity is updated.")

	var jobLines []string
	for i, j := range m.jobNames {
		if i >= 5 {
			jobLines = append(jobLines, StyleMuted.Render(fmt.Sprintf("  … and %d more", len(m.jobNames)-5)))
			break
		}
		jobLines = append(jobLines, StyleMuted.Render("  • "+j))
	}
	jobList := strings.Join(jobLines, "\n")

	confirmStyle := StyleMuted
	cancelStyle := StyleWarning.Bold(true)
	if m.focused == 0 {
		confirmStyle = StyleSuccess.Bold(true)
		cancelStyle = StyleMuted
	}

	confirmBtn := confirmStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Confirm")
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, confirmBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: select  esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Left,
		icon+"  "+title, "", body, jobList, "", warn, "", buttons, "", hint)
}

// ─── 7. DeleteModal (source/destination with job list) ────────────────────────

// DeleteModal is the delete confirmation for sources/destinations that have jobs.
// Confirm = delete,  Cancel = go back.
type DeleteModal struct {
	entityName string
	entityKind string // "source" or "destination"
	jobNames   []string
	focused    int // 0=Delete, 1=Cancel
}

func NewDeleteModal(entityName, entityKind string, jobNames []string) *DeleteModal {
	return &DeleteModal{entityName: entityName, entityKind: entityKind, jobNames: jobNames, focused: 1}
}

func (m *DeleteModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "y", "Y":
		return m, ModalActionConfirm
	case "n", "N":
		return m, ModalActionCancel
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *DeleteModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *DeleteModal) NeedsSpinner() bool                    { return false }

func (m *DeleteModal) View() string {
	icon := StyleError.Bold(true).Render("⚠")
	kind := m.entityKind
	if len(kind) > 0 {
		kind = strings.ToUpper(kind[:1]) + kind[1:]
	}
	title := StyleError.Bold(true).Render(fmt.Sprintf("Delete %s", kind))
	body := StyleNormal.Render(fmt.Sprintf(
		"Deleting '%s' will disable all associated jobs. Are you sure?", m.entityName))

	var jobLines []string
	for i, j := range m.jobNames {
		if i >= 5 {
			jobLines = append(jobLines, StyleMuted.Render(fmt.Sprintf("  … and %d more", len(m.jobNames)-5)))
			break
		}
		jobLines = append(jobLines, StyleMuted.Render("  • "+j))
	}
	jobList := strings.Join(jobLines, "\n")

	deleteStyle := StyleMuted
	cancelStyle := StyleMuted
	if m.focused == 0 {
		deleteStyle = StyleError.Bold(true)
	} else {
		cancelStyle = StyleNormal.Bold(true)
	}

	deleteBtn := deleteStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Delete")
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, deleteBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("y/n: choose  ←→/tab: move  enter: confirm  esc: cancel")

	parts := []string{icon + "  " + title, "", body}
	if len(jobLines) > 0 {
		parts = append(parts, "", StyleMuted.Render("Affected jobs:"), jobList)
	}
	parts = append(parts, "", buttons, "", hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ─── 8. DeleteJobModal ────────────────────────────────────────────────────────

// DeleteJobModal confirms deletion of a job.
// Confirm = delete,  Cancel = go back.
type DeleteJobModal struct {
	jobName string
	focused int // 0=Delete, 1=Cancel
}

func NewDeleteJobModal(jobName string) *DeleteJobModal {
	return &DeleteJobModal{jobName: jobName, focused: 1}
}

func (m *DeleteJobModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "y", "Y":
		return m, ModalActionConfirm
	case "n", "N":
		return m, ModalActionCancel
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *DeleteJobModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *DeleteJobModal) NeedsSpinner() bool                    { return false }

func (m *DeleteJobModal) View() string {
	icon := StyleWarning.Bold(true).Render("⚠")
	title := StyleTitle.Render("Delete Job")
	body := StyleNormal.Render(fmt.Sprintf("Are you sure you want to delete job '%s'?", m.jobName))
	sub := StyleWarning.Render("This will remove all run history and cannot be undone.")

	deleteStyle := StyleMuted
	cancelStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		deleteStyle = StyleError.Bold(true)
		cancelStyle = StyleMuted
	}

	deleteBtn := deleteStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Delete")
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, deleteBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("y/n: choose  ←→/tab: move  enter: confirm  esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, sub, "", buttons, "", hint)
}

// ─── 9. DestinationDatabaseModal ─────────────────────────────────────────────

// DestinationDatabaseFormat is dynamic (namespace-based) or custom (single name).
type DestinationDatabaseFormat int

const (
	DestDBFormatDynamic DestinationDatabaseFormat = iota
	DestDBFormatCustom
)

// DestinationDatabaseModal edits the S3 folder / Iceberg DB name.
// Confirm = save,  Cancel = close.
type DestinationDatabaseModal struct {
	isIceberg    bool
	format       DestinationDatabaseFormat
	namespaces   []string
	input        textinput.Model
	focusField   int // 0=format radio, 1=input, 2=buttons
	buttonFocus  int // 0=Save, 1=Close
}

func NewDestinationDatabaseModal(isIceberg bool, current string, namespaces []string) *DestinationDatabaseModal {
	ti := textinput.New()
	ti.Placeholder = "my_prefix"
	ti.SetValue(current)
	ti.Width = 40
	ti.Focus()

	return &DestinationDatabaseModal{
		isIceberg:  isIceberg,
		namespaces: namespaces,
		input:      ti,
		focusField: 1,
	}
}

func (m *DestinationDatabaseModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "tab":
		m.focusField = (m.focusField + 1) % 3
		if m.focusField == 1 {
			m.input.Focus()
		} else {
			m.input.Blur()
		}
	case "up", "k":
		if m.focusField == 0 {
			if m.format == DestDBFormatCustom {
				m.format = DestDBFormatDynamic
			}
		}
	case "down", "j":
		if m.focusField == 0 {
			if m.format == DestDBFormatDynamic {
				m.format = DestDBFormatCustom
			}
		}
	case "left", "h":
		if m.focusField == 2 {
			m.buttonFocus = 0
		}
	case "right", "l":
		if m.focusField == 2 {
			m.buttonFocus = 1
		}
	case "enter":
		if m.focusField == 2 {
			if m.buttonFocus == 0 {
				return m, ModalActionConfirm
			}
			return m, ModalActionCancel
		}
		if m.focusField < 2 {
			m.focusField++
			if m.focusField == 1 {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
		}
	}

	if m.focusField == 1 {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		_ = cmd
	}
	return m, ModalActionNone
}

func (m *DestinationDatabaseModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *DestinationDatabaseModal) NeedsSpinner() bool                    { return false }

func (m *DestinationDatabaseModal) Value() string { return m.input.Value() }
func (m *DestinationDatabaseModal) Format() DestinationDatabaseFormat { return m.format }

func (m *DestinationDatabaseModal) View() string {
	var titleStr string
	if m.isIceberg {
		titleStr = "Iceberg Database Name"
	} else {
		titleStr = "S3 Folder Name"
	}
	title := StyleTitle.Render(titleStr)

	// Format radio
	dynLabel := "  Dynamic (Default)"
	cusLabel := "  Custom"
	if m.format == DestDBFormatDynamic {
		dynLabel = StyleCyan().Render("● Dynamic (Default)")
		cusLabel = StyleMuted.Render("○ Custom")
	} else {
		dynLabel = StyleMuted.Render("○ Dynamic (Default)")
		cusLabel = StyleCyan().Render("● Custom")
	}

	formatSection := ""
	if m.focusField == 0 {
		formatSection = StylePanelFocused.Render(
			lipgloss.JoinVertical(lipgloss.Left, "Format:", dynLabel, cusLabel))
	} else {
		formatSection = StylePanel.Render(
			lipgloss.JoinVertical(lipgloss.Left, "Format:", dynLabel, cusLabel))
	}

	// Input
	inputLabel := "Prefix / Name:"
	if m.focusField == 1 {
		inputLabel = StyleCyan().Render("Prefix / Name:")
	} else {
		inputLabel = StyleMuted.Render("Prefix / Name:")
	}
	inputRow := inputLabel + "  " + m.input.View()

	// Preview
	prefix := m.input.Value()
	var previewLines []string
	if m.format == DestDBFormatDynamic && len(m.namespaces) > 0 {
		for i, ns := range m.namespaces {
			if i >= 3 {
				previewLines = append(previewLines, StyleMuted.Render("  …"))
				break
			}
			previewLines = append(previewLines, StyleMuted.Render(
				fmt.Sprintf("  %s_%s", prefix, ns)))
		}
	} else if m.format == DestDBFormatCustom {
		if prefix != "" {
			previewLines = []string{StyleMuted.Render("  " + prefix)}
		}
	}

	previewSection := ""
	if len(previewLines) > 0 {
		previewSection = "\nPreview:\n" + strings.Join(previewLines, "\n")
	}

	// Buttons
	saveStyle := StyleMuted
	closeStyle := StyleMuted
	if m.focusField == 2 {
		if m.buttonFocus == 0 {
			saveStyle = StyleSuccess.Bold(true)
		} else {
			closeStyle = StyleNormal.Bold(true)
		}
	}
	saveBtn := saveStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Save Changes")
	closeBtn := closeStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Close")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, saveBtn, "  ", closeBtn)

	hint := StyleMuted.Render("tab: move  ↑↓: format  enter: next/confirm  esc: close")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", formatSection, "", inputRow, previewSection, "", buttons, "", hint)
}

// StyleCyan returns a lipgloss style with the OLake cyan foreground.
func StyleCyan() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
}

// ─── 10. ClearDestinationModal ────────────────────────────────────────────────

// ClearDestinationModal warns before clearing all job destination data.
// Confirm = proceed to ClearDataModal (double-confirm),  Cancel = close.
type ClearDestinationModal struct {
	jobName string
	focused int // 0=Confirm, 1=Cancel
	loading bool
}

func NewClearDestinationModal(jobName string) *ClearDestinationModal {
	return &ClearDestinationModal{jobName: jobName, focused: 1}
}

func (m *ClearDestinationModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	if m.loading {
		return m, ModalActionNone
	}
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *ClearDestinationModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *ClearDestinationModal) NeedsSpinner() bool                    { return false }

func (m *ClearDestinationModal) View() string {
	icon := StyleError.Bold(true).Render("⚠")
	title := StyleTitle.Render("Clear Destination")
	body := StyleError.Render("This will erase ALL data that was synced by this job in the destination.")
	jobLine := StyleNormal.Render("Job: " + m.jobName)
	warn := StyleWarning.Render("This action cannot be undone.")

	confirmStyle := StyleMuted
	cancelStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		confirmStyle = StyleError.Bold(true)
		cancelStyle = StyleMuted
	}

	confirmBtn := confirmStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Clear Data")
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, confirmBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: confirm  esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, jobLine, warn, "", buttons, "", hint)
}

// ─── 11. ClearDataModal ───────────────────────────────────────────────────────

// ClearDataModal is the second confirmation for the clear destination flow.
// Confirm = execute,  Cancel = go back.
type ClearDataModal struct {
	focused int // 0=Clear, 1=Cancel
}

func NewClearDataModal() *ClearDataModal {
	return &ClearDataModal{focused: 1}
}

func (m *ClearDataModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *ClearDataModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *ClearDataModal) NeedsSpinner() bool                    { return false }

func (m *ClearDataModal) View() string {
	icon := StyleError.Bold(true).Render("⚠")
	title := StyleError.Bold(true).Render("Final Confirmation")
	body := StyleNormal.Render("Clear data will delete ALL data in your job.")
	warn := StyleWarning.Render("This is your last chance to cancel. Are you absolutely sure?")

	clearStyle := StyleMuted
	cancelStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		clearStyle = StyleError.Bold(true)
		cancelStyle = StyleMuted
	}

	clearBtn := clearStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Clear Data")
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, clearBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: confirm  esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, warn, "", buttons, "", hint)
}

// ─── 12. SpecFailedModal ─────────────────────────────────────────────────────

// SpecFailedModal is shown when the connector spec (schema) fails to load.
// Confirm = Try Again,  Cancel = Close.
type SpecFailedModal struct {
	entityKind string // "Source" or "Destination"
	errMsg     string
	focused    int // 0=Close, 1=Try Again
}

func NewSpecFailedModal(entityKind, errMsg string) *SpecFailedModal {
	return &SpecFailedModal{entityKind: entityKind, errMsg: errMsg, focused: 1}
}

func (m *SpecFailedModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 1 {
			return m, ModalActionConfirm // Try Again
		}
		return m, ModalActionCancel // Close
	}
	return m, ModalActionNone
}

func (m *SpecFailedModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *SpecFailedModal) NeedsSpinner() bool                    { return false }

func (m *SpecFailedModal) View() string {
	icon := StyleError.Bold(true).Render("✗")
	title := StyleError.Bold(true).Render(m.entityKind + " Spec Load Failed")
	body := StyleNormal.Render("Failed to load the connector specification.")
	errLine := StyleError.Render(m.errMsg)

	closeStyle := StyleMuted
	retryStyle := StyleMuted
	if m.focused == 0 {
		closeStyle = StyleNormal.Bold(true)
	} else {
		retryStyle = StyleSuccess.Bold(true)
	}

	closeBtn := closeStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Close")
	retryBtn := retryStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Try Again")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, closeBtn, "  ", retryBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: select  esc: close")
	return lipgloss.JoinVertical(lipgloss.Left, icon+"  "+title, "", body, errLine, "", buttons, "", hint)
}

// ─── 13. StreamDifferenceModal ────────────────────────────────────────────────

// StreamDifference holds the added/removed stream names for display.
type StreamDifference struct {
	Added   []string
	Removed []string
}

// StreamDifferenceModal shows which streams changed before saving a job edit.
// Confirm = "Confirm and Finish",  Cancel = go back.
type StreamDifferenceModal struct {
	diff    StreamDifference
	focused int // 0=Confirm, 1=Cancel
	loading bool
}

func NewStreamDifferenceModal(diff StreamDifference) *StreamDifferenceModal {
	return &StreamDifferenceModal{diff: diff, focused: 1}
}

func (m *StreamDifferenceModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	if m.loading {
		return m, ModalActionNone
	}
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			m.loading = true
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *StreamDifferenceModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *StreamDifferenceModal) NeedsSpinner() bool                    { return false }

func (m *StreamDifferenceModal) View() string {
	icon := StyleWarning.Bold(true).Render("⚠")
	title := StyleTitle.Render("Stream Changes Detected")
	question := StyleNormal.Render("Are you sure you want to continue?")
	warn := StyleWarning.Render("Any ongoing sync will be auto-cancelled.")

	var diffLines []string
	if len(m.diff.Added) > 0 {
		diffLines = append(diffLines, StyleSuccess.Render("  Added streams:"))
		for i, s := range m.diff.Added {
			if i >= 5 {
				diffLines = append(diffLines, StyleMuted.Render(fmt.Sprintf("    … and %d more", len(m.diff.Added)-5)))
				break
			}
			diffLines = append(diffLines, StyleSuccess.Render("    + "+s))
		}
	}
	if len(m.diff.Removed) > 0 {
		diffLines = append(diffLines, StyleError.Render("  Removed streams:"))
		for i, s := range m.diff.Removed {
			if i >= 5 {
				diffLines = append(diffLines, StyleMuted.Render(fmt.Sprintf("    … and %d more", len(m.diff.Removed)-5)))
				break
			}
			diffLines = append(diffLines, StyleError.Render("    - "+s))
		}
	}
	diffSection := strings.Join(diffLines, "\n")

	confirmLabel := "Confirm and Finish"
	if m.loading {
		confirmLabel = "Saving…"
	}

	confirmStyle := StyleMuted
	cancelStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		confirmStyle = StyleSuccess.Bold(true)
		cancelStyle = StyleMuted
	}

	confirmBtn := confirmStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render(confirmLabel)
	cancelBtn := cancelStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, confirmBtn, "  ", cancelBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: confirm  esc: cancel")

	parts := []string{icon + "  " + title, "", question}
	if diffSection != "" {
		parts = append(parts, "", diffSection)
	}
	parts = append(parts, "", warn, "", buttons, "", hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ─── 14. IngestionModeChangeModal ─────────────────────────────────────────────

// IngestionModeChangeModal confirms switching all streams' ingestion mode.
// Confirm = apply,  Cancel = close.
type IngestionModeChangeModal struct {
	mode    string // "Upsert" or "Append"
	focused int    // 0=Confirm, 1=Close
}

func NewIngestionModeChangeModal(mode string) *IngestionModeChangeModal {
	return &IngestionModeChangeModal{mode: mode, focused: 1}
}

func (m *IngestionModeChangeModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc":
		return m, ModalActionCancel
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm
		}
		return m, ModalActionCancel
	}
	return m, ModalActionNone
}

func (m *IngestionModeChangeModal) Tick(_ time.Time) (Modal, ModalAction) {
	return m, ModalActionNone
}
func (m *IngestionModeChangeModal) NeedsSpinner() bool { return false }

func (m *IngestionModeChangeModal) View() string {
	title := StyleTitle.Render(fmt.Sprintf("Switch to %s for all tables?", m.mode))
	body := StyleNormal.Render(fmt.Sprintf(
		"All tables will be switched to %s mode.", m.mode))
	sub := StyleMuted.Render("You can still change mode for individual tables afterwards.")

	confirmStyle := StyleMuted
	closeStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		confirmStyle = StyleSuccess.Bold(true)
		closeStyle = StyleMuted
	}

	confirmBtn := confirmStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Confirm")
	closeBtn := closeStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Close")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, confirmBtn, "  ", closeBtn)

	hint := StyleMuted.Render("←→/tab: move  enter: select  esc: close")
	return lipgloss.JoinVertical(lipgloss.Center, title, "", body, sub, "", buttons, "", hint)
}

// ─── 15. ResetStreamsModal ────────────────────────────────────────────────────

// ResetStreamsModal warns that leaving the streams step will lose all progress.
// Confirm = leave (ModalActionConfirm),  Cancel = stay.
type ResetStreamsModal struct {
	focused int // 0=Yes Leave, 1=No
}

func NewResetStreamsModal() *ResetStreamsModal {
	return &ResetStreamsModal{focused: 1}
}

func (m *ResetStreamsModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc", "n", "N":
		return m, ModalActionCancel
	case "y", "Y":
		return m, ModalActionConfirm
	case "left", "h":
		m.focused = 0
	case "right", "l":
		m.focused = 1
	case "tab":
		m.focused = (m.focused + 1) % 2
	case "enter":
		if m.focused == 0 {
			return m, ModalActionConfirm // Yes, Leave
		}
		return m, ModalActionCancel // No
	}
	return m, ModalActionNone
}

func (m *ResetStreamsModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *ResetStreamsModal) NeedsSpinner() bool                    { return false }

func (m *ResetStreamsModal) View() string {
	icon := StyleWarning.Bold(true).Render("⚠")
	title := StyleTitle.Render("Your changes will not be saved")
	body := StyleNormal.Render("Leaving this page will lose all your progress & changes.")
	question := StyleWarning.Render("Are you sure you want to leave?")

	yesStyle := StyleMuted
	noStyle := StyleNormal.Bold(true)
	if m.focused == 0 {
		yesStyle = StyleError.Bold(true)
		noStyle = StyleMuted
	}

	yesBtn := yesStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Yes, Leave")
	noBtn := noStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("No")
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "  ", noBtn)

	hint := StyleMuted.Render("y/n: choose  ←→/tab: move  enter: confirm  esc: stay")
	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, question, "", buttons, "", hint)
}

// ─── 16. StreamEditDisabledModal ──────────────────────────────────────────────

// StreamEditDisabledContext specifies where the disabled modal is triggered from.
type StreamEditDisabledContext int

const (
	StreamEditDisabledFromJobSettings StreamEditDisabledContext = iota
	StreamEditDisabledFromJobEdit
)

// StreamEditDisabledModal informs the user that stream editing is currently disabled.
// Confirm = "Go back to Jobs".
type StreamEditDisabledModal struct {
	from StreamEditDisabledContext
}

func NewStreamEditDisabledModal(from StreamEditDisabledContext) *StreamEditDisabledModal {
	return &StreamEditDisabledModal{from: from}
}

func (m *StreamEditDisabledModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc", "enter", "q":
		return m, ModalActionConfirm // Go back to Jobs
	}
	return m, ModalActionNone
}

func (m *StreamEditDisabledModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *StreamEditDisabledModal) NeedsSpinner() bool                    { return false }

func (m *StreamEditDisabledModal) View() string {
	icon := StyleRunning.Bold(true).Render("ℹ")
	title := StyleTitle.Render("Editing Disabled")

	var msg string
	switch m.from {
	case StreamEditDisabledFromJobSettings:
		msg = "Job Settings Edit is disabled while a destination clear is in progress.\nPlease wait for the clear operation to complete."
	case StreamEditDisabledFromJobEdit:
		msg = "Stream editing is disabled while a destination clear is in progress.\nPlease wait for the clear operation to complete."
	}

	body := StyleNormal.Render(msg)
	btn := StyleSuccess.Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Go back to Jobs")
	hint := StyleMuted.Render("enter/esc: go back to Jobs")

	return lipgloss.JoinVertical(lipgloss.Center, icon, title, "", body, "", btn, "", hint)
}

// ─── 17. UpdatesModal ────────────────────────────────────────────────────────

// UpdateCategory is one release category in the updates modal.
type UpdateCategory struct {
	Name     string
	HasNew   bool
	Releases []UpdateRelease
}

// UpdateRelease is a single release entry.
type UpdateRelease struct {
	Version string
	Date    string
	Tags    []string
	Content string // Markdown-like content
}

// UpdatesModal shows OLake release notes and update notifications.
// Cancel / Esc = close.
type UpdatesModal struct {
	categories   []UpdateCategory
	catIdx       int // selected category
	releaseIdx   int // selected release within category
	expanded     bool
	focusPane    int // 0=category list, 1=releases
}

func NewUpdatesModal(categories []UpdateCategory) *UpdatesModal {
	return &UpdatesModal{categories: categories}
}

func (m *UpdatesModal) Update(msg tea.KeyMsg) (Modal, ModalAction) {
	switch msg.String() {
	case "esc", "q":
		return m, ModalActionCancel

	case "tab":
		m.focusPane = (m.focusPane + 1) % 2

	case "up", "k":
		if m.focusPane == 0 {
			if m.catIdx > 0 {
				m.catIdx--
				m.releaseIdx = 0
				m.expanded = false
			}
		} else {
			if m.releaseIdx > 0 {
				m.releaseIdx--
				m.expanded = false
			}
		}

	case "down", "j":
		if m.focusPane == 0 {
			if m.catIdx < len(m.categories)-1 {
				m.catIdx++
				m.releaseIdx = 0
				m.expanded = false
			}
		} else {
			cat := m.currentCategory()
			if cat != nil && m.releaseIdx < len(cat.Releases)-1 {
				m.releaseIdx++
				m.expanded = false
			}
		}

	case "enter", " ":
		if m.focusPane == 1 {
			m.expanded = !m.expanded
		}
	}
	return m, ModalActionNone
}

func (m *UpdatesModal) Tick(_ time.Time) (Modal, ModalAction) { return m, ModalActionNone }
func (m *UpdatesModal) NeedsSpinner() bool                    { return false }

func (m *UpdatesModal) currentCategory() *UpdateCategory {
	if m.catIdx >= 0 && m.catIdx < len(m.categories) {
		return &m.categories[m.catIdx]
	}
	return nil
}

func (m *UpdatesModal) View() string {
	title := StyleTitle.Render("OLake Updates")

	if len(m.categories) == 0 {
		return lipgloss.JoinVertical(lipgloss.Center,
			title, "", StyleMuted.Render("No updates available."), "", StyleMuted.Render("esc: close"))
	}

	// Left: category list
	var catLines []string
	for i, c := range m.categories {
		label := c.Name
		prefix := "  "
		style := StyleMuted
		if i == m.catIdx {
			style = StyleSelected
			prefix = "▶ "
		}
		newDot := ""
		if c.HasNew {
			newDot = StyleError.Render(" ●")
		}
		catLines = append(catLines, prefix+style.Render(label)+newDot)
	}

	catListStyle := StylePanel
	if m.focusPane == 0 {
		catListStyle = StylePanelFocused
	}
	catPanel := catListStyle.Width(22).Render(
		StyleMuted.Render("Categories") + "\n" + strings.Join(catLines, "\n"))

	// Right: releases for selected category
	cat := m.currentCategory()
	var releaseContent string
	if cat == nil || len(cat.Releases) == 0 {
		releaseContent = StyleMuted.Render("No releases in this category.")
	} else {
		var rLines []string
		for i, r := range cat.Releases {
			isSelected := i == m.releaseIdx
			prefix := "  "
			titleStyle := StyleNormal
			if isSelected && m.focusPane == 1 {
				prefix = "▶ "
				titleStyle = StyleSelected
			}

			// Tags
			var tagParts []string
			for j, t := range r.Tags {
				if j >= 2 {
					break
				}
				tagParts = append(tagParts, StyleCyan().Render("["+t+"]"))
			}
			tags := strings.Join(tagParts, " ")

			header := prefix + titleStyle.Render(r.Version) + "  " + StyleMuted.Render(r.Date) + "  " + tags

			if isSelected && m.expanded && r.Content != "" {
				rLines = append(rLines, header)
				rLines = append(rLines, StyleMuted.Render("  "+strings.ReplaceAll(r.Content, "\n", "\n  ")))
				rLines = append(rLines, "")
			} else {
				rLines = append(rLines, header)
			}
		}
		releaseContent = strings.Join(rLines, "\n")
	}

	releaseStyle := StylePanel
	if m.focusPane == 1 {
		releaseStyle = StylePanelFocused
	}
	releasePanel := releaseStyle.Width(44).Render(
		StyleMuted.Render("Releases") + "\n" + releaseContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, catPanel, "  ", releasePanel)

	hint := StyleMuted.Render("tab: switch pane  ↑↓/j/k: navigate  enter: expand  esc: close")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", hint)
}

// ─── ModalState ───────────────────────────────────────────────────────────────

// ModalState holds the currently active modal (if any).
// The app model stores exactly one *ModalState.
type ModalState struct {
	current Modal
	// spinnerCmd is the pending spinner tick command, if the modal needs one.
	spinnerCmd tea.Cmd
}

// Active reports whether a modal is currently shown.
func (s *ModalState) Active() bool {
	return s != nil && s.current != nil
}

// Show sets the active modal and, if needed, starts the spinner tick chain.
func (s *ModalState) Show(m Modal) tea.Cmd {
	s.current = m
	if m.NeedsSpinner() {
		return ModalSpinnerTick(100 * time.Millisecond)
	}
	return nil
}

// Dismiss clears the active modal.
func (s *ModalState) Dismiss() {
	s.current = nil
}

// Current returns the active modal or nil.
func (s *ModalState) Current() Modal {
	if s == nil {
		return nil
	}
	return s.current
}

// HandleKey routes a key press to the active modal.
// Returns the resulting action and a follow-up command (e.g. continue spinner).
func (s *ModalState) HandleKey(msg tea.KeyMsg) (ModalAction, tea.Cmd) {
	if s == nil || s.current == nil {
		return ModalActionNone, nil
	}
	updated, action := s.current.Update(msg)
	s.current = updated
	if action == ModalActionDismiss || action == ModalActionConfirm || action == ModalActionCancel || action == ModalActionAlt {
		return action, nil
	}
	return ModalActionNone, nil
}

// HandleTick routes a tick message to the active modal.
func (s *ModalState) HandleTick(t ModalTickMsg) (ModalAction, tea.Cmd) {
	if s == nil || s.current == nil {
		return ModalActionNone, nil
	}
	updated, action := s.current.Tick(t.T)
	s.current = updated
	if action == ModalActionDismiss {
		return ModalActionDismiss, nil
	}
	if s.current != nil && s.current.NeedsSpinner() {
		return action, ModalSpinnerTick(100 * time.Millisecond)
	}
	return action, nil
}

// View renders the modal overlay or "" if no modal is active.
func (s *ModalState) View(width, height int) string {
	if s == nil || s.current == nil {
		return ""
	}
	return ModalOverlay(s.current, width, height)
}
