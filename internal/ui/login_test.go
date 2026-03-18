package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLoginModel_Init(t *testing.T) {
	m := NewLoginModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a blink command, got nil")
	}
}

func TestLoginModel_DefaultFocus(t *testing.T) {
	m := NewLoginModel()
	// By default focus is on the username field (index 0).
	if m.focused != 0 {
		t.Errorf("expected focused=0, got %d", m.focused)
	}
}

func TestLoginModel_TabSwitchesField(t *testing.T) {
	m := NewLoginModel()

	// Tab key should move focus from username → password.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m2.focused != 1 {
		t.Errorf("after Tab, expected focused=1, got %d", m2.focused)
	}

	// Tab again wraps back to username.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m3.focused != 0 {
		t.Errorf("after second Tab, expected focused=0, got %d", m3.focused)
	}
}

func TestLoginModel_DownSwitchesField(t *testing.T) {
	m := NewLoginModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.focused != 1 {
		t.Errorf("after Down, expected focused=1, got %d", m2.focused)
	}
}

func TestLoginModel_ShiftTabSwitchesBack(t *testing.T) {
	m := NewLoginModel()
	// Move to password first.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// ShiftTab back to username.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m3.focused != 0 {
		t.Errorf("after ShiftTab, expected focused=0, got %d", m3.focused)
	}
}

func TestLoginModel_EnterOnUsernameAdvancesToPassword(t *testing.T) {
	m := NewLoginModel()

	// Pressing Enter on the username field should move focus to password,
	// not emit a LoginMsg.
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m2.focused != 1 {
		t.Errorf("Enter on username should focus password; got focused=%d", m2.focused)
	}

	// The cmd should be a blink command, not a LoginMsg emitter — so executing
	// it must not return a LoginMsg.
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(LoginMsg); ok {
			t.Error("Enter on username field should NOT emit LoginMsg; should just advance focus")
		}
	}
}

func TestLoginModel_EnterOnPasswordEmitsLoginMsg(t *testing.T) {
	m := NewLoginModel()
	// Move to password field.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Set a value in the username input before submitting.
	// We simulate a char type by updating the input.
	m.usernameInput.SetValue("admin")
	m.passwordInput.SetValue("secret")

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m2

	if cmd == nil {
		t.Fatal("Enter on password should return a command")
	}

	msg := cmd()
	loginMsg, ok := msg.(LoginMsg)
	if !ok {
		t.Fatalf("expected LoginMsg, got %T", msg)
	}
	if loginMsg.Username != "admin" {
		t.Errorf("expected Username=admin, got %q", loginMsg.Username)
	}
	if loginMsg.Password != "secret" {
		t.Errorf("expected Password=secret, got %q", loginMsg.Password)
	}
}

func TestLoginModel_SetError(t *testing.T) {
	m := NewLoginModel()
	m.SetError("invalid credentials")

	view := m.View()
	if !strings.Contains(view, "invalid credentials") {
		t.Error("View() should display the error message")
	}
}

func TestLoginModel_ErrorClearedOnKeyPress(t *testing.T) {
	m := NewLoginModel()
	m.SetError("some error")

	// Any key press should clear the error.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m2.err != "" {
		t.Errorf("error should be cleared after key press, got %q", m2.err)
	}
}

func TestLoginModel_ViewContainsExpectedFields(t *testing.T) {
	m := NewLoginModel()
	view := m.View()

	checks := []string{"Username", "Password"}
	for _, check := range checks {
		if !strings.Contains(view, check) {
			t.Errorf("View() should contain %q", check)
		}
	}
}

func TestLoginModel_SetSize(t *testing.T) {
	m := NewLoginModel()
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("SetSize: want (120,40), got (%d,%d)", m.width, m.height)
	}
}

func TestLoginModel_ViewWithSize(t *testing.T) {
	m := NewLoginModel()
	m.SetSize(120, 40)
	view := m.View()
	// With a size set, the view uses lipgloss.Place — it should still contain
	// the OLake logo.
	if !strings.Contains(view, "OLake") {
		t.Error("View() with size should still contain OLake branding")
	}
}
