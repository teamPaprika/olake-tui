package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all application key bindings.
type KeyMap struct {
	// Navigation
	Tab      key.Binding
	ShiftTab key.Binding
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Tab4     key.Binding
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Esc      key.Binding
	Back     key.Binding

	// List actions
	Add     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Test    key.Binding
	Sync    key.Binding
	Cancel  key.Binding
	Logs    key.Binding
	Pause   key.Binding

	// Job operations
	Settings     key.Binding // capital S — open job settings
	Detail       key.Binding // enter — open job detail
	ClearDest    key.Binding // clear destination action
	Tab5         key.Binding // '5' — system settings tab

	// App
	Quit key.Binding
	Help key.Binding
}

// DefaultKeyMap returns the standard key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
		Tab1:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "jobs")),
		Tab2:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "sources")),
		Tab3:     key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "destinations")),
		Tab4:     key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "settings")),
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Esc:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),

		Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Test:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "test connection")),
		Sync:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync now")),
		Cancel:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cancel run")),
		Logs:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
		Pause:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause/resume")),

		Settings:  key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "job settings")),
		Detail:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "job detail")),
		ClearDest: key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "clear destination")),
		Tab5:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "system settings")),

		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}
