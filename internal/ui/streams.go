package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// StreamEntry holds display + config data for a single discovered stream.
type StreamEntry struct {
	Namespace    string
	Name         string
	SyncModes    []string // supported sync modes
	CursorFields []string // available cursor fields
	// User-chosen config
	SyncMode    string
	CursorField string
	Normalize   bool
	Selected    bool
}

// fullName returns "namespace.name" (or just "name" if namespace is empty).
func (s StreamEntry) fullName() string {
	if s.Namespace != "" {
		return s.Namespace + "." + s.Name
	}
	return s.Name
}

// StreamConfigPopup is a modal for per-stream configuration.
type StreamConfigPopup struct {
	stream      *StreamEntry // pointer into parent slice
	syncIdx     int          // cursor in sync-mode list
	syncModes   []string
	cursorInput textinput.Model
	normalize   bool
	focusField  int // 0=syncMode 1=cursorField 2=normalize 3=confirm
}

func newStreamConfigPopup(s *StreamEntry) StreamConfigPopup {
	modes := s.SyncModes
	if len(modes) == 0 {
		modes = []string{"full_refresh", "incremental", "cdc", "strict_cdc"}
	}

	idx := 0
	for i, m := range modes {
		if m == s.SyncMode {
			idx = i
			break
		}
	}

	ci := textinput.New()
	ci.Placeholder = "cursor field (optional)"
	ci.SetValue(s.CursorField)
	ci.Width = 30

	return StreamConfigPopup{
		stream:      s,
		syncIdx:     idx,
		syncModes:   modes,
		cursorInput: ci,
		normalize:   s.Normalize,
	}
}

// syncModeLabel maps api values to display names.
func syncModeLabel(m string) string {
	switch m {
	case "full_refresh":
		return "Full Refresh"
	case "incremental":
		return "Full Refresh + Incremental"
	case "cdc":
		return "Full Refresh + CDC"
	case "strict_cdc":
		return "CDC Only"
	default:
		return m
	}
}

// handleKey processes keyboard input in the popup.
// Returns true when the popup should close (enter on confirm).
func (p *StreamConfigPopup) handleKey(msg tea.KeyMsg) (close bool) {
	switch msg.String() {
	case "tab", "down", "j":
		p.focusField = (p.focusField + 1) % 4
		if p.focusField == 1 {
			p.cursorInput.Focus()
		} else {
			p.cursorInput.Blur()
		}
	case "shift+tab", "up", "k":
		p.focusField = (p.focusField + 3) % 4
		if p.focusField == 1 {
			p.cursorInput.Focus()
		} else {
			p.cursorInput.Blur()
		}
	case "left", "h":
		if p.focusField == 0 && p.syncIdx > 0 {
			p.syncIdx--
		}
	case "right", "l":
		if p.focusField == 0 && p.syncIdx < len(p.syncModes)-1 {
			p.syncIdx++
		}
	case " ":
		if p.focusField == 2 {
			p.normalize = !p.normalize
		}
	case "enter":
		if p.focusField == 3 {
			// Apply changes back to stream
			p.stream.SyncMode = p.syncModes[p.syncIdx]
			p.stream.CursorField = p.cursorInput.Value()
			p.stream.Normalize = p.normalize
			return true
		}
	}
	if p.focusField == 1 {
		p.cursorInput, _ = p.cursorInput.Update(msg)
	}
	return false
}

func (p *StreamConfigPopup) view(width, height int) string {
	title := StyleTitle.Render("Configure Stream: " + p.stream.fullName())

	// Sync mode row
	smLabel := StyleBold.Render("Sync Mode:")
	var smParts []string
	for i, m := range p.syncModes {
		label := syncModeLabel(m)
		if i == p.syncIdx {
			smParts = append(smParts, StyleTabActive.Render("["+label+"]"))
		} else {
			smParts = append(smParts, StyleMuted.Render(" "+label+" "))
		}
	}
	smLine := smLabel + " " + strings.Join(smParts, " ")
	if p.focusField == 0 {
		smLine = StylePanelFocused.Render(smLine)
	}

	// Cursor field
	cfLabel := StyleBold.Render("Cursor Field:")
	cfLine := cfLabel + " " + p.cursorInput.View()
	if p.focusField == 1 {
		cfLine = StylePanelFocused.Render(cfLine)
	}

	// Normalize toggle
	normCheck := "[ ]"
	if p.normalize {
		normCheck = StyleSuccess.Render("[x]")
	}
	normLine := StyleBold.Render("Normalize: ") + normCheck + StyleMuted.Render(" (toggle with space)")
	if p.focusField == 2 {
		normLine = StylePanelFocused.Render(normLine)
	}

	// Confirm button
	confirmStyle := StyleMuted.Bold(true)
	if p.focusField == 3 {
		confirmStyle = StyleSuccess.Bold(true)
	}
	confirmBtn := confirmStyle.Padding(0, 2).Border(lipgloss.RoundedBorder()).Render("Apply")

	hint := StyleHelp.Render("tab/↑↓: move  ←→: change sync mode  space: toggle  enter: apply  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, "",
		smLine, "",
		cfLine, "",
		normLine, "",
		lipgloss.NewStyle().Align(lipgloss.Center).Render(confirmBtn), "",
		hint,
	)

	box := StylePanelFocused.Width(70).Padding(1, 2).Render(content)
	if width > 0 && height > 0 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

// ─── StreamsModel ─────────────────────────────────────────────────────────────

// StreamsModel is the full, reusable stream selection component.
type StreamsModel struct {
	streams    []StreamEntry
	filtered   []int // indices into streams after search filter
	cursor     int   // index into filtered
	search     textinput.Model
	searching  bool
	popup      *StreamConfigPopup
	loading    bool
	loadErr    string
	spinner    spinner.Model
	width      int
	height     int
	scrollTop  int // first visible row
	visibleMax int // computed visible rows
}

// NewStreamsModel creates a new streams selection component.
func NewStreamsModel() StreamsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)

	si := textinput.New()
	si.Placeholder = "search streams..."
	si.Width = 30

	return StreamsModel{spinner: s, search: si}
}

// SetStreams populates the stream list from discovered data.
func (m *StreamsModel) SetStreams(infos []service.StreamInfo) {
	m.streams = make([]StreamEntry, len(infos))
	for i, info := range infos {
		mode := "full_refresh"
		if len(info.SyncModes) > 0 {
			mode = info.SyncModes[0]
		}
		m.streams[i] = StreamEntry{
			Namespace:    info.Namespace,
			Name:         info.Name,
			SyncModes:    info.SyncModes,
			CursorFields: info.CursorFields,
			SyncMode:     mode,
			Selected:     false,
		}
	}
	m.loading = false
	m.loadErr = ""
	m.cursor = 0
	m.scrollTop = 0
	m.rebuildFilter()
}

// SetLoading sets the loading state.
func (m *StreamsModel) SetLoading(v bool) { m.loading = v }

// SetError sets an error message.
func (m *StreamsModel) SetError(err string) {
	m.loading = false
	m.loadErr = err
}

// SetSize updates terminal dimensions.
func (m *StreamsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.visibleMax = h - 10
	if m.visibleMax < 3 {
		m.visibleMax = 3
	}
}

// GetSelectedConfigs returns the user-selected stream configurations.
func (m *StreamsModel) GetSelectedConfigs() []service.StreamConfig {
	var configs []service.StreamConfig
	for _, s := range m.streams {
		if s.Selected {
			configs = append(configs, service.StreamConfig{
				Namespace:   s.Namespace,
				Name:        s.Name,
				SyncMode:    s.SyncMode,
				CursorField: s.CursorField,
				Normalize:   s.Normalize,
				Selected:    true,
			})
		}
	}
	return configs
}

// SelectedCount returns number of selected streams.
func (m *StreamsModel) SelectedCount() int {
	n := 0
	for _, s := range m.streams {
		if s.Selected {
			n++
		}
	}
	return n
}

// TotalCount returns total number of streams.
func (m *StreamsModel) TotalCount() int { return len(m.streams) }

// HasPopup returns true if the config popup is open.
func (m *StreamsModel) HasPopup() bool { return m.popup != nil }

// rebuildFilter rebuilds the filtered index list based on search text.
func (m *StreamsModel) rebuildFilter() {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.filtered = nil
	for i, s := range m.streams {
		if q == "" || strings.Contains(strings.ToLower(s.fullName()), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.clampScroll()
}

func (m *StreamsModel) clampScroll() {
	if m.cursor < m.scrollTop {
		m.scrollTop = m.cursor
	}
	if m.visibleMax > 0 && m.cursor >= m.scrollTop+m.visibleMax {
		m.scrollTop = m.cursor - m.visibleMax + 1
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Init implements part of tea.Model.
func (m StreamsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles keyboard events.
func (m StreamsModel) Update(msg tea.Msg) (StreamsModel, tea.Cmd) {
	// Delegate to popup when open.
	if m.popup != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.popup = nil
				return m, nil
			}
			if m.popup.handleKey(msg) {
				m.popup = nil
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "esc", "enter":
				m.searching = false
				m.search.Blur()
				m.rebuildFilter()
			default:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				m.rebuildFilter()
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.searching = true
			m.search.Focus()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.clampScroll()
			}
		case " ":
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.streams[idx].Selected = !m.streams[idx].Selected
			}
		case "a":
			// Toggle all: if any selected → deselect all; else select all
			anySelected := false
			for _, s := range m.streams {
				if s.Selected {
					anySelected = true
					break
				}
			}
			for i := range m.streams {
				m.streams[i].Selected = !anySelected
			}
		case "enter":
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				popup := newStreamConfigPopup(&m.streams[idx])
				m.popup = &popup
			}
		}
	}
	return m, nil
}

// View renders the stream selection component.
func (m StreamsModel) View() string {
	if m.popup != nil {
		return m.popup.view(m.width, m.height)
	}

	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Stream Selection"),
			"",
			m.spinner.View()+" Discovering streams...",
		)
	}
	if m.loadErr != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Stream Selection"),
			"",
			StyleError.Render("Error: "+m.loadErr),
			"",
			StyleMuted.Render("Check source configuration and try again."),
		)
	}

	// Header
	countLine := StyleMuted.Render(fmt.Sprintf("%d/%d streams selected", m.SelectedCount(), m.TotalCount()))
	searchLine := m.search.View()
	if !m.searching {
		searchLine = StyleMuted.Render("/ to search")
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		StyleTitle.Render("Streams  "),
		countLine,
		"    ",
		searchLine,
	)

	if len(m.filtered) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			StyleMuted.Render("No streams match your filter."),
		)
	}

	// Group by namespace for display
	type nsGroup struct {
		ns    string
		idxs  []int // indices into m.filtered
	}
	nsMap := map[string]*nsGroup{}
	var nsOrder []string
	for _, fi := range m.filtered {
		s := m.streams[fi]
		if _, ok := nsMap[s.Namespace]; !ok {
			nsMap[s.Namespace] = &nsGroup{ns: s.Namespace}
			nsOrder = append(nsOrder, s.Namespace)
		}
		nsMap[s.Namespace].idxs = append(nsMap[s.Namespace].idxs, fi)
	}
	sort.Strings(nsOrder)

	// Build flat row list with namespace headers
	type rowItem struct {
		filteredIdx int    // position in m.filtered (-1 = ns header)
		display     string
		isHeader    bool
	}
	var rows []rowItem
	filteredPos := map[int]int{} // filtered idx → row position
	rowPos := 0
	for _, ns := range nsOrder {
		grp := nsMap[ns]
		nsLabel := ns
		if nsLabel == "" {
			nsLabel = "(default)"
		}
		rows = append(rows, rowItem{filteredIdx: -1, display: StyleMuted.Render("  ┌─ "+nsLabel), isHeader: true})
		rowPos++
		for _, fi := range grp.idxs {
			filteredPos[fi] = rowPos
			rows = append(rows, rowItem{filteredIdx: fi})
			rowPos++
		}
	}

	// Visible window
	vm := m.visibleMax
	if vm <= 0 {
		vm = 20
	}

	// Find the display row of the cursor entry
	cursorFi := -1
	if m.cursor < len(m.filtered) {
		cursorFi = m.filtered[m.cursor]
	}
	cursorRow := 0
	if cursorFi >= 0 {
		if rp, ok := filteredPos[cursorFi]; ok {
			cursorRow = rp
		}
	}

	// Compute scroll offset in row terms
	scrollRows := m.scrollTop
	if cursorRow < scrollRows {
		scrollRows = cursorRow
	}
	if cursorRow >= scrollRows+vm {
		scrollRows = cursorRow - vm + 1
	}

	var lines []string
	displayed := 0
	for ri, row := range rows {
		_ = ri
		if displayed < scrollRows {
			displayed++
			continue
		}
		if len(lines) >= vm {
			break
		}
		if row.isHeader {
			lines = append(lines, row.display)
		} else {
			fi := row.filteredIdx
			s := m.streams[fi]

			check := "[ ]"
			if s.Selected {
				check = StyleSuccess.Render("[x]")
			}
			name := s.Name
			if len(name) > 30 {
				name = name[:29] + "…"
			}
			sm := syncModeLabel(s.SyncMode)
			line := fmt.Sprintf("  %s %-30s  %s", check, name, StyleMuted.Render("sync: "+sm))

			if fi == cursorFi {
				lines = append(lines, StyleSelected.Render("> ")+strings.TrimPrefix(line, "  "))
			} else {
				lines = append(lines, line)
			}
		}
		displayed++
	}

	list := strings.Join(lines, "\n")

	var hint string
	if m.searching {
		hint = "type to filter  esc/enter: done"
	} else {
		hint = "↑↓/j/k:move  space:toggle  a:all/none  enter:configure  /:search  esc:back"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header, "",
		list, "",
		StyleHelp.Render(hint),
	)
}
