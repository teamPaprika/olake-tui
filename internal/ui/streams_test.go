package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func sampleStreams() []service.StreamInfo {
	return []service.StreamInfo{
		{Namespace: "public", Name: "users", SyncModes: []string{"full_refresh", "cdc"}},
		{Namespace: "public", Name: "orders", SyncModes: []string{"full_refresh"}},
		{Namespace: "public", Name: "products", SyncModes: []string{"full_refresh", "incremental"}},
		{Namespace: "audit", Name: "events", SyncModes: []string{"cdc"}},
	}
}

func newStreamsWithData() StreamsModel {
	m := NewStreamsModel()
	m.SetStreams(sampleStreams())
	m.SetSize(120, 40)
	return m
}

func sendKey(m StreamsModel, key string) StreamsModel {
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return m2
}

// ─── Initialization ───────────────────────────────────────────────────────────

func TestStreamsModel_Init(t *testing.T) {
	m := NewStreamsModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a spinner tick command")
	}
}

func TestStreamsModel_SetStreams(t *testing.T) {
	m := NewStreamsModel()
	m.SetStreams(sampleStreams())

	if m.TotalCount() != 4 {
		t.Errorf("want 4 streams, got %d", m.TotalCount())
	}
	if m.SelectedCount() != 0 {
		t.Errorf("newly loaded streams should have 0 selected, got %d", m.SelectedCount())
	}
	if m.loading {
		t.Error("loading should be false after SetStreams")
	}
}

// ─── Toggle individual stream ─────────────────────────────────────────────────

func TestStreamsModel_ToggleStream(t *testing.T) {
	m := newStreamsWithData()

	// Space toggles the current stream.
	m = sendKey(m, " ")
	if m.SelectedCount() != 1 {
		t.Errorf("after one toggle, want 1 selected, got %d", m.SelectedCount())
	}

	// Toggle again deselects.
	m = sendKey(m, " ")
	if m.SelectedCount() != 0 {
		t.Errorf("after second toggle, want 0 selected, got %d", m.SelectedCount())
	}
}

// ─── Select all / deselect all ────────────────────────────────────────────────

func TestStreamsModel_SelectAll(t *testing.T) {
	m := newStreamsWithData()

	// 'a' selects all when none are selected.
	m = sendKey(m, "a")
	if m.SelectedCount() != m.TotalCount() {
		t.Errorf("after 'a', want all %d selected, got %d", m.TotalCount(), m.SelectedCount())
	}
}

func TestStreamsModel_DeselectAll(t *testing.T) {
	m := newStreamsWithData()

	// Select all first.
	m = sendKey(m, "a")
	// 'a' again deselects all because anySelected=true.
	m = sendKey(m, "a")
	if m.SelectedCount() != 0 {
		t.Errorf("after second 'a', want 0 selected, got %d", m.SelectedCount())
	}
}

func TestStreamsModel_SelectAllThenPartialDeselectThenToggleAll(t *testing.T) {
	m := newStreamsWithData()

	m = sendKey(m, "a") // select all
	m = sendKey(m, " ") // deselect cursor (first item)
	// Now at least one is still selected, so 'a' should deselect all.
	m = sendKey(m, "a")
	if m.SelectedCount() != 0 {
		t.Errorf("want 0 selected after deselect-all, got %d", m.SelectedCount())
	}
}

// ─── Navigation ───────────────────────────────────────────────────────────────

func TestStreamsModel_CursorMovement(t *testing.T) {
	m := newStreamsWithData()

	start := m.cursor
	m = sendKey(m, "j")
	if m.cursor <= start {
		t.Errorf("cursor should move down after 'j', was %d now %d", start, m.cursor)
	}

	m = sendKey(m, "k")
	if m.cursor != start {
		t.Errorf("cursor should move back up after 'k', want %d got %d", start, m.cursor)
	}
}

func TestStreamsModel_CursorDoesNotGoNegative(t *testing.T) {
	m := newStreamsWithData()
	m.cursor = 0

	m = sendKey(m, "k") // up at top should not panic
	if m.cursor != 0 {
		t.Errorf("cursor at 0, pressing up should stay at 0, got %d", m.cursor)
	}
}

// ─── Search / filter ──────────────────────────────────────────────────────────

func TestStreamsModel_SearchFilter(t *testing.T) {
	m := newStreamsWithData()

	// Enter search mode.
	m = sendKey(m, "/")
	if !m.searching {
		t.Fatal("expected searching=true after '/'")
	}

	// Type "user" — only "users" stream should match.
	for _, ch := range "user" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	if len(m.filtered) != 1 {
		t.Errorf("filter 'user' should yield 1 result, got %d", len(m.filtered))
	}

	// Exit search mode.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m2.searching {
		t.Error("searching should be false after Esc")
	}
}

func TestStreamsModel_SearchFilterNoMatch(t *testing.T) {
	m := newStreamsWithData()
	m = sendKey(m, "/")
	for _, ch := range "zzznomatch" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if len(m.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.filtered))
	}
}

func TestStreamsModel_SearchEnterExits(t *testing.T) {
	m := newStreamsWithData()
	m = sendKey(m, "/")
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m2.searching {
		t.Error("enter should exit search mode")
	}
}

// ─── Stream count display ─────────────────────────────────────────────────────

func TestStreamsModel_ViewShowsCount(t *testing.T) {
	m := newStreamsWithData()
	m = sendKey(m, "a") // select all

	view := m.View()
	// View should contain the count of selected / total streams.
	total := m.TotalCount()
	if !strings.Contains(view, "4") {
		t.Errorf("view should contain total stream count (%d), view was:\n%s", total, view)
	}
}

// ─── Loading and error states ─────────────────────────────────────────────────

func TestStreamsModel_LoadingState(t *testing.T) {
	m := NewStreamsModel()
	m.SetLoading(true)

	view := m.View()
	if !strings.Contains(view, "Discovering") {
		t.Error("loading view should contain 'Discovering'")
	}
}

func TestStreamsModel_ErrorState(t *testing.T) {
	m := NewStreamsModel()
	m.SetError("connection refused")

	view := m.View()
	if !strings.Contains(view, "connection refused") {
		t.Error("error view should display the error message")
	}
}

// ─── GetSelectedConfigs ───────────────────────────────────────────────────────

func TestStreamsModel_GetSelectedConfigs(t *testing.T) {
	m := newStreamsWithData()

	// Select first two streams.
	m = sendKey(m, " ")
	m = sendKey(m, "j")
	m = sendKey(m, " ")

	configs := m.GetSelectedConfigs()
	if len(configs) != 2 {
		t.Errorf("want 2 selected configs, got %d", len(configs))
	}
	for _, c := range configs {
		if !c.Selected {
			t.Error("all returned configs should have Selected=true")
		}
	}
}

func TestStreamsModel_GetSelectedConfigsEmpty(t *testing.T) {
	m := newStreamsWithData()
	configs := m.GetSelectedConfigs()
	if len(configs) != 0 {
		t.Errorf("want 0 configs when nothing selected, got %d", len(configs))
	}
}

// ─── Popup ────────────────────────────────────────────────────────────────────

func TestStreamsModel_PopupOpenAndClose(t *testing.T) {
	m := newStreamsWithData()

	// Enter opens popup.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m2.HasPopup() {
		t.Error("Enter should open config popup")
	}

	// Esc closes popup.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m3.HasPopup() {
		t.Error("Esc should close config popup")
	}
}
