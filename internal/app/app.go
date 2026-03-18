// Package app implements the main Bubble Tea application model.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
	"github.com/datazip-inc/olake-tui/internal/ui"
)

// Screen identifies which screen is currently displayed.
type Screen int

const (
	ScreenLogin Screen = iota
	ScreenJobs
	ScreenSources
	ScreenDestinations
	ScreenSettings
	ScreenJobLogs
	ScreenConfirm
)

// Tab identifiers (for main navigation).
type Tab int

const (
	TabJobs Tab = iota
	TabSources
	TabDestinations
	TabSettings
)

// --- Async result message types ---

type msgLoginDone struct{ err error }
type msgJobsLoaded struct {
	jobs []service.Job
	err  error
}
type msgSourcesLoaded struct {
	sources []service.Source
	err     error
}
type msgDestsLoaded struct {
	dests []service.Destination
	err   error
}
type msgJobDeleted struct{ err error }
type msgSourceDeleted struct{ err error }
type msgDestDeleted struct{ err error }
type msgSyncTriggered struct{ err error }
type msgCancelDone struct{ err error }
type msgActivateDone struct{ err error }
type msgLogsLoaded struct {
	logs []service.LogEntry
	err  error
}
type msgToastExpired struct{}

// confirmContext identifies what action a confirmation dialog is for.
type confirmContext int

const (
	confirmNone confirmContext = iota
	confirmDeleteJob
	confirmDeleteSource
	confirmDeleteDest
	confirmSync
	confirmCancel
)

// Model is the root Bubble Tea model.
type Model struct {
	svc     *service.Manager
	keys    KeyMap
	screen  Screen
	tab     Tab
	width   int
	height  int

	// Sub-models
	login        ui.LoginModel
	jobs         ui.JobsModel
	sources      ui.SourcesModel
	destinations ui.DestinationsModel
	logs         *ui.JobLogsModel
	confirm      ui.ConfirmModel
	confirmCtx   confirmContext
	confirmID    int

	// Data
	jobList   []service.Job
	srcList   []service.Source
	dstList   []service.Destination

	// Toast notification
	toast      string
	toastError bool

	// Auth state
	authenticated bool
	username      string
}

// New creates the root application model.
func New(svc *service.Manager) Model {
	return Model{
		svc:          svc,
		keys:         DefaultKeyMap(),
		screen:       ScreenLogin,
		tab:          TabJobs,
		login:        ui.NewLoginModel(),
		jobs:         ui.NewJobsModel(),
		sources:      ui.NewSourcesModel(),
		destinations: ui.NewDestinationsModel(),
	}
}

// Init is called on program start.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.login.Init(),
		m.jobs.Init(),
	)
}

// showToast shows a transient notification that auto-clears after 3 seconds.
func showToast(msg string, isErr bool) tea.Cmd {
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return msgToastExpired{}
	})
}

// Update handles all messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.login.SetSize(m.width, m.height)
		m.sources.SetSize(m.width, m.height)
		m.destinations.SetSize(m.width, m.height)
		m.jobs.SetSize(m.width, m.height)
		if m.logs != nil {
			m.logs.SetSize(m.width, m.height)
		}

	// ---------- Toast expiry ----------
	case msgToastExpired:
		m.toast = ""

	// ---------- Login ----------
	case ui.LoginMsg:
		m.login.SetError("")
		return m, func() tea.Msg {
			err := m.svc.Login(msg.Username, msg.Password)
			return msgLoginDone{err: err}
		}

	case msgLoginDone:
		if msg.err != nil {
			m.login.SetError(msg.err.Error())
		} else {
			m.authenticated = true
			m.username = m.svc.Username()
			m.screen = ScreenJobs
			m.tab = TabJobs
			m.jobs.SetLoading(true)
			cmds = append(cmds, m.loadJobs())
		}

	// ---------- Jobs ----------
	case msgJobsLoaded:
		if msg.err != nil {
			m.jobs.SetError(msg.err.Error())
		} else {
			m.jobList = msg.jobs
			m.jobs.SetJobs(msg.jobs)
		}

	case msgJobDeleted:
		if msg.err != nil {
			m.toast = "Delete failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Job deleted"
			m.toastError = false
			m.jobs.SetLoading(true)
			cmds = append(cmds, m.loadJobs())
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	case msgSyncTriggered:
		if msg.err != nil {
			m.toast = "Sync failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Sync triggered!"
			m.toastError = false
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	case msgCancelDone:
		if msg.err != nil {
			m.toast = "Cancel failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Job cancelled"
			m.toastError = false
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	case msgActivateDone:
		if msg.err != nil {
			m.toast = "Failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Job updated"
			m.toastError = false
			m.jobs.SetLoading(true)
			cmds = append(cmds, m.loadJobs())
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	// ---------- Sources ----------
	case msgSourcesLoaded:
		if msg.err != nil {
			m.sources.SetError(msg.err.Error())
		} else {
			m.srcList = msg.sources
			m.sources.SetSources(msg.sources)
		}

	case msgSourceDeleted:
		if msg.err != nil {
			m.toast = "Delete failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Source deleted"
			m.toastError = false
			m.sources.SetLoading(true)
			cmds = append(cmds, m.loadSources())
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	// ---------- Destinations ----------
	case msgDestsLoaded:
		if msg.err != nil {
			m.destinations.SetError(msg.err.Error())
		} else {
			m.dstList = msg.dests
			m.destinations.SetDestinations(msg.dests)
		}

	case msgDestDeleted:
		if msg.err != nil {
			m.toast = "Delete failed: " + msg.err.Error()
			m.toastError = true
		} else {
			m.toast = "Destination deleted"
			m.toastError = false
			m.destinations.SetLoading(true)
			cmds = append(cmds, m.loadDests())
		}
		cmds = append(cmds, showToast(m.toast, m.toastError))

	// ---------- Logs ----------
	case msgLogsLoaded:
		if m.logs != nil {
			if msg.err != nil {
				m.logs.SetError(msg.err.Error())
			} else {
				m.logs.SetLogs(msg.logs)
			}
		}

	// ---------- Confirm dialog result ----------
	case confirmResult:
		return m, m.handleConfirmResult(msg.yes)

	// ---------- Key events ----------
	case tea.KeyMsg:
		return m.handleKey(msg, cmds)
	}

	// Delegate updates to sub-models
	if len(cmds) == 0 {
		cmds = append(cmds, m.delegateUpdate(msg))
	}

	return m, tea.Batch(cmds...)
}

// confirmResult is sent by the confirm dialog.
type confirmResult struct{ yes bool }

// handleKey routes key events based on current screen.
func (m Model) handleKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "q" || msg.String() == "ctrl+c" {
		if m.screen != ScreenLogin && m.screen != ScreenConfirm {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Esc from logs → back to jobs
	if msg.Type == tea.KeyEsc {
		if m.screen == ScreenJobLogs {
			m.screen = ScreenJobs
			m.logs = nil
			return m, nil
		}
		if m.screen == ScreenConfirm {
			m.screen = m.screenBeforeConfirm()
			return m, nil
		}
	}

	// Confirm dialog
	if m.screen == ScreenConfirm {
		action := m.confirm.HandleKey(msg.String())
		switch action {
		case ui.ConfirmYes:
			m.screen = m.screenBeforeConfirm()
			return m, func() tea.Msg { return confirmResult{yes: true} }
		case ui.ConfirmNo:
			m.screen = m.screenBeforeConfirm()
			return m, nil
		}
		return m, nil
	}

	// Login screen
	if m.screen == ScreenLogin {
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd
	}

	// Log viewer
	if m.screen == ScreenJobLogs {
		var cmd tea.Cmd
		logs, cmd := m.logs.Update(msg)
		m.logs = &logs
		return m, cmd
	}

	// Tab switching
	switch msg.String() {
	case "1":
		return m.switchTab(TabJobs), nil
	case "2":
		return m.switchTab(TabSources), nil
	case "3":
		return m.switchTab(TabDestinations), nil
	case "4":
		return m.switchTab(TabSettings), nil
	case "tab":
		next := (int(m.tab) + 1) % 4
		return m.switchTab(Tab(next)), nil
	}

	// Tab-specific actions
	return m.handleTabKey(msg)
}

// handleTabKey routes key events within the current tab.
func (m Model) handleTabKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.tab {
	case TabJobs:
		switch msg.String() {
		case "r":
			m.jobs.SetLoading(true)
			return m, m.loadJobs()
		case "s":
			if j := m.jobs.SelectedJob(); j != nil {
				return m.showConfirm("Sync Job", fmt.Sprintf("Trigger sync for '%s'?", j.Name), confirmSync, j.ID), nil
			}
		case "c":
			if j := m.jobs.SelectedJob(); j != nil {
				return m.showConfirm("Cancel Job", fmt.Sprintf("Cancel running sync for '%s'?", j.Name), confirmCancel, j.ID), nil
			}
		case "d":
			if j := m.jobs.SelectedJob(); j != nil {
				return m.showConfirm("Delete Job", fmt.Sprintf("Delete job '%s'? This cannot be undone.", j.Name), confirmDeleteJob, j.ID), nil
			}
		case "p":
			if j := m.jobs.SelectedJob(); j != nil {
				jID := j.ID
				activate := !j.Activate
				return m, func() tea.Msg {
					err := m.svc.ActivateJob(jID, activate)
					return msgActivateDone{err: err}
				}
			}
		case "l":
			if j := m.jobs.SelectedJob(); j != nil {
				return m.openLogs(j.ID), nil
			}
		default:
			var cmd tea.Cmd
			m.jobs, cmd = m.jobs.Update(msg)
			return m, cmd
		}

	case TabSources:
		switch msg.String() {
		case "r":
			m.sources.SetLoading(true)
			return m, m.loadSources()
		case "d":
			if s := m.sources.SelectedSource(); s != nil {
				return m.showConfirm("Delete Source", fmt.Sprintf("Delete source '%s'?", s.Name), confirmDeleteSource, s.ID), nil
			}
		default:
			var cmd tea.Cmd
			m.sources, cmd = m.sources.Update(msg)
			return m, cmd
		}

	case TabDestinations:
		switch msg.String() {
		case "r":
			m.destinations.SetLoading(true)
			return m, m.loadDests()
		case "d":
			if d := m.destinations.SelectedDestination(); d != nil {
				return m.showConfirm("Delete Destination", fmt.Sprintf("Delete destination '%s'?", d.Name), confirmDeleteDest, d.ID), nil
			}
		default:
			var cmd tea.Cmd
			m.destinations, cmd = m.destinations.Update(msg)
			return m, cmd
		}

	case TabSettings:
		// Settings tab is read-only for now
	}

	return m, nil
}

// handleConfirmResult processes the confirmed action.
func (m *Model) handleConfirmResult(yes bool) tea.Cmd {
	if !yes {
		return nil
	}
	id := m.confirmID
	switch m.confirmCtx {
	case confirmDeleteJob:
		return func() tea.Msg {
			err := m.svc.DeleteJob(id)
			return msgJobDeleted{err: err}
		}
	case confirmDeleteSource:
		return func() tea.Msg {
			err := m.svc.DeleteSource(id)
			return msgSourceDeleted{err: err}
		}
	case confirmDeleteDest:
		return func() tea.Msg {
			err := m.svc.DeleteDestination(id)
			return msgDestDeleted{err: err}
		}
	case confirmSync:
		return func() tea.Msg {
			err := m.svc.TriggerSync(id)
			return msgSyncTriggered{err: err}
		}
	case confirmCancel:
		return func() tea.Msg {
			err := m.svc.CancelJob(id)
			return msgCancelDone{err: err}
		}
	}
	return nil
}

// showConfirm transitions to the confirmation dialog.
func (m Model) showConfirm(title, msg string, ctx confirmContext, id int) Model {
	m.confirm = ui.NewConfirmModel(title, msg)
	m.confirmCtx = ctx
	m.confirmID = id
	m.screen = ScreenConfirm
	return m
}

// screenBeforeConfirm returns the screen to go back to after confirm.
func (m Model) screenBeforeConfirm() Screen {
	switch m.tab {
	case TabSources:
		return ScreenSources
	case TabDestinations:
		return ScreenDestinations
	case TabSettings:
		return ScreenSettings
	default:
		return ScreenJobs
	}
}

// switchTab switches to the given tab and loads its data if needed.
func (m Model) switchTab(t Tab) Model {
	m.tab = t
	switch t {
	case TabJobs:
		m.screen = ScreenJobs
		if len(m.jobList) == 0 {
			m.jobs.SetLoading(true)
		}
	case TabSources:
		m.screen = ScreenSources
		if len(m.srcList) == 0 {
			m.sources.SetLoading(true)
		}
	case TabDestinations:
		m.screen = ScreenDestinations
		if len(m.dstList) == 0 {
			m.destinations.SetLoading(true)
		}
	case TabSettings:
		m.screen = ScreenSettings
	}
	return m
}

// openLogs loads task list and then opens the log viewer.
func (m Model) openLogs(jobID int) Model {
	// Get tasks, then open first task's logs.
	// For v1, open a stub log viewer and load on demand.
	logModel := ui.NewJobLogsModel(jobID, "latest", "", m.width, m.height)
	m.logs = &logModel
	m.screen = ScreenJobLogs
	return m
}

// delegateUpdate forwards messages to sub-models.
func (m Model) delegateUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.screen {
	case ScreenJobs:
		m.jobs, cmd = m.jobs.Update(msg)
	case ScreenSources:
		m.sources, cmd = m.sources.Update(msg)
	case ScreenDestinations:
		m.destinations, cmd = m.destinations.Update(msg)
	case ScreenJobLogs:
		if m.logs != nil {
			logs, c := m.logs.Update(msg)
			m.logs = &logs
			cmd = c
		}
	}
	return cmd
}

// --- Async loaders ---

func (m Model) loadJobs() tea.Cmd {
	return func() tea.Msg {
		jobs, err := m.svc.ListJobs()
		return msgJobsLoaded{jobs: jobs, err: err}
	}
}

func (m Model) loadSources() tea.Cmd {
	return func() tea.Msg {
		sources, err := m.svc.ListSources()
		return msgSourcesLoaded{sources: sources, err: err}
	}
}

func (m Model) loadDests() tea.Cmd {
	return func() tea.Msg {
		dests, err := m.svc.ListDestinations()
		return msgDestsLoaded{dests: dests, err: err}
	}
}

// --- View ---

// View renders the full TUI.
func (m Model) View() string {
	if m.screen == ScreenLogin {
		return m.login.View()
	}

	// Render the confirm overlay on top of current screen
	if m.screen == ScreenConfirm {
		return m.confirm.View(m.width, m.height)
	}

	// Compute available height for content (subtract header + status bar)
	headerH := 6
	statusH := 1
	contentH := m.height - headerH - statusH
	if contentH < 5 {
		contentH = 5
	}

	header := m.renderHeader()
	content := m.renderContent(contentH)
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

// renderHeader renders the top navigation bar.
func (m Model) renderHeader() string {
	// Dashboard stats
	stats := ui.ComputeDashboardStats(m.srcList, m.dstList, m.jobList)
	dash := ui.RenderDashboard(stats, m.username)

	// Tabs
	tabs := m.renderTabs()

	return lipgloss.JoinVertical(lipgloss.Left, dash, "", tabs, "")
}

// renderTabs renders the tab navigation bar.
func (m Model) renderTabs() string {
	tabs := []struct {
		id    Tab
		label string
		key   string
	}{
		{TabJobs, "Jobs", "1"},
		{TabSources, "Sources", "2"},
		{TabDestinations, "Destinations", "3"},
		{TabSettings, "Settings", "4"},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.key, t.label)
		if m.tab == t.id {
			parts = append(parts, ui.StyleTabActive.Render(label))
		} else {
			parts = append(parts, ui.StyleTabInactive.Render(label))
		}
	}

	return strings.Join(parts, "  ")
}

// renderContent renders the main content area for the current tab.
func (m Model) renderContent(height int) string {
	_ = height // can be used for viewport sizing

	if m.screen == ScreenJobLogs && m.logs != nil {
		return m.logs.View()
	}

	switch m.tab {
	case TabJobs:
		return m.jobs.View()
	case TabSources:
		return m.sources.View()
	case TabDestinations:
		return m.destinations.View()
	case TabSettings:
		return m.renderSettings()
	}
	return ""
}

// renderSettings renders the settings screen placeholder.
func (m Model) renderSettings() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		ui.StyleTitle.Render("Settings"),
		"",
		ui.StyleMuted.Render("Webhook Alert URL: (not loaded)"),
		"",
		ui.StyleHelp.Render("r: refresh settings"),
	)
}

// renderStatusBar renders the bottom hint bar.
func (m Model) renderStatusBar() string {
	var hint string
	switch m.screen {
	case ScreenJobs:
		hint = "1-4:tabs  s:sync  c:cancel  l:logs  p:pause  d:delete  r:refresh  q:quit"
	case ScreenSources:
		hint = "1-4:tabs  a:add  e:edit  d:delete  t:test  r:refresh  q:quit"
	case ScreenDestinations:
		hint = "1-4:tabs  a:add  e:edit  d:delete  t:test  r:refresh  q:quit"
	case ScreenJobLogs:
		hint = "↑↓/pgup/pgdn:scroll  esc:back  q:quit"
	default:
		hint = "1-4:tabs  q:quit"
	}

	status := ui.StyleStatusBar.Width(m.width).Render(hint)

	if m.toast != "" {
		var toastView string
		if m.toastError {
			toastView = ui.StyleToastError.Render(" " + m.toast + " ")
		} else {
			toastView = ui.StyleToastSuccess.Render(" " + m.toast + " ")
		}
		return lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", toastView)
	}

	return status
}
