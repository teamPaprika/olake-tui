package ui

import (
	"strings"
	"testing"
)

// ─── NewConfirmModel ──────────────────────────────────────────────────────────

func TestConfirmModel_NewModel(t *testing.T) {
	c := NewConfirmModel("Delete Source", "Are you sure you want to delete this source?")
	if c.Title != "Delete Source" {
		t.Errorf("title mismatch: want %q, got %q", "Delete Source", c.Title)
	}
	if c.Message != "Are you sure you want to delete this source?" {
		t.Errorf("message mismatch: got %q", c.Message)
	}
	if c.focused != 0 {
		t.Errorf("default focus should be Yes (0), got %d", c.focused)
	}
}

// ─── Y / N shortcut keys ─────────────────────────────────────────────────────

func TestConfirmModel_YKeyConfirms(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("y")
	if action != ConfirmYes {
		t.Errorf("'y' should return ConfirmYes, got %v", action)
	}
}

func TestConfirmModel_UpperYKeyConfirms(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("Y")
	if action != ConfirmYes {
		t.Errorf("'Y' should return ConfirmYes, got %v", action)
	}
}

func TestConfirmModel_NKeyCancels(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("n")
	if action != ConfirmNo {
		t.Errorf("'n' should return ConfirmNo, got %v", action)
	}
}

func TestConfirmModel_UpperNKeyCancels(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("N")
	if action != ConfirmNo {
		t.Errorf("'N' should return ConfirmNo, got %v", action)
	}
}

func TestConfirmModel_EscCancels(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("esc")
	if action != ConfirmNo {
		t.Errorf("'esc' should return ConfirmNo, got %v", action)
	}
}

// ─── Enter key ───────────────────────────────────────────────────────────────

func TestConfirmModel_EnterOnYesConfirms(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 0 // Yes is focused by default

	action := c.HandleKey("enter")
	if action != ConfirmYes {
		t.Errorf("Enter with Yes focused should return ConfirmYes, got %v", action)
	}
}

func TestConfirmModel_EnterOnNoCancels(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 1 // Move focus to No first

	action := c.HandleKey("enter")
	if action != ConfirmNo {
		t.Errorf("Enter with No focused should return ConfirmNo, got %v", action)
	}
}

// ─── Navigation ───────────────────────────────────────────────────────────────

func TestConfirmModel_LeftKeyFocusesYes(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 1 // start at No

	action := c.HandleKey("left")
	if c.focused != 0 {
		t.Errorf("left should move focus to Yes (0), got %d", c.focused)
	}
	if action != ConfirmNone {
		t.Errorf("navigation should return ConfirmNone, got %v", action)
	}
}

func TestConfirmModel_RightKeyFocusesNo(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 0 // start at Yes

	action := c.HandleKey("right")
	if c.focused != 1 {
		t.Errorf("right should move focus to No (1), got %d", c.focused)
	}
	if action != ConfirmNone {
		t.Errorf("navigation should return ConfirmNone, got %v", action)
	}
}

func TestConfirmModel_HKeyFocusesYes(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 1
	c.HandleKey("h")
	if c.focused != 0 {
		t.Errorf("'h' should focus Yes, got %d", c.focused)
	}
}

func TestConfirmModel_LKeyFocusesNo(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	c.focused = 0
	c.HandleKey("l")
	if c.focused != 1 {
		t.Errorf("'l' should focus No, got %d", c.focused)
	}
}

// ─── Unknown keys ─────────────────────────────────────────────────────────────

func TestConfirmModel_UnknownKeyReturnsNone(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	action := c.HandleKey("x")
	if action != ConfirmNone {
		t.Errorf("unknown key should return ConfirmNone, got %v", action)
	}
}

// ─── View rendering ───────────────────────────────────────────────────────────

func TestConfirmModel_ViewContainsTitleAndMessage(t *testing.T) {
	title := "Delete Job"
	msg := "This action cannot be undone."
	c := NewConfirmModel(title, msg)
	view := c.View(0, 0)

	if !strings.Contains(view, title) {
		t.Errorf("view should contain title %q", title)
	}
	if !strings.Contains(view, msg) {
		t.Errorf("view should contain message %q", msg)
	}
}

func TestConfirmModel_ViewContainsButtons(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	view := c.View(0, 0)

	if !strings.Contains(view, "Yes") {
		t.Error("view should contain 'Yes' button")
	}
	if !strings.Contains(view, "No") {
		t.Error("view should contain 'No' button")
	}
}

func TestConfirmModel_ViewWithSize(t *testing.T) {
	c := NewConfirmModel("Test", "msg")
	view := c.View(120, 40)

	// With dimensions provided, lipgloss.Place is used — still should have content.
	if !strings.Contains(view, "Yes") {
		t.Error("View with size should still contain 'Yes'")
	}
}

// ─── ConfirmAction values ─────────────────────────────────────────────────────

func TestConfirmAction_Constants(t *testing.T) {
	if ConfirmNone == ConfirmYes {
		t.Error("ConfirmNone and ConfirmYes must be distinct")
	}
	if ConfirmNone == ConfirmNo {
		t.Error("ConfirmNone and ConfirmNo must be distinct")
	}
	if ConfirmYes == ConfirmNo {
		t.Error("ConfirmYes and ConfirmNo must be distinct")
	}
}
