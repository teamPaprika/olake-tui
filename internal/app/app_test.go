// Package app_test provides integration tests for the root Bubble Tea model.
// All tests use MockService — no external dependencies required.
package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/service"
	"github.com/datazip-inc/olake-tui/internal/ui"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// newTestModel creates a Model backed by a fresh MockService.
func newTestModel() (Model, *service.MockService) {
	mock := service.NewMockService()
	mock.Sources = []service.Source{
		{ID: 1, Name: "pg-prod", Type: "postgres", Version: "1.0", CreatedAt: time.Now()},
		{ID: 2, Name: "mongo-dev", Type: "mongodb", Version: "1.0", CreatedAt: time.Now()},
	}
	mock.Destinations = []service.Destination{
		{ID: 10, Name: "s3-bucket", Type: "s3", Version: "1.0", CreatedAt: time.Now()},
	}
	mock.Jobs = []service.Job{
		{
			ID:          100,
			Name:        "nightly-sync",
			Frequency:   "0 0 * * *",
			Activate:    true,
			Source:      service.JobConnector{ID: 1, Name: "pg-prod", Type: "postgres"},
			Destination: service.JobConnector{ID: 10, Name: "s3-bucket", Type: "s3"},
		},
	}
	return New(mock, "test"), mock
}

// update applies a message to the model and returns the updated model + any command.
func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	newModel, cmd := m.Update(msg)
	return newModel.(Model), cmd
}

// runCmd executes a command and returns the message it produces (nil-safe).
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// ─── Initialization ───────────────────────────────────────────────────────────

func TestNew_InitialScreen(t *testing.T) {
	m, _ := newTestModel()
	if m.screen != ScreenLogin {
		t.Errorf("initial screen should be ScreenLogin, got %v", m.screen)
	}
}

func TestNew_NotAuthenticated(t *testing.T) {
	m, _ := newTestModel()
	if m.authenticated {
		t.Error("new model should not be authenticated")
	}
}

func TestNew_Init(t *testing.T) {
	m, _ := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

// ─── Login flow ───────────────────────────────────────────────────────────────

func TestLoginFlow_Success(t *testing.T) {
	m, _ := newTestModel()

	// Simulate the LoginMsg that the LoginModel emits.
	m2, cmd := update(m, ui.LoginMsg{Username: "admin", Password: "pass"})

	if cmd == nil {
		t.Fatal("login should produce an async command")
	}

	// Execute the async login command.
	loginResult := runCmd(cmd)
	m3, _ := update(m2, loginResult)

	if !m3.authenticated {
		t.Error("model should be authenticated after successful login")
	}
	if m3.screen != ScreenJobs {
		t.Errorf("after login, screen should be ScreenJobs, got %v", m3.screen)
	}
}

func TestLoginFlow_Failure(t *testing.T) {
	m, mock := newTestModel()
	mock.LoginErr = errTest("invalid credentials")

	m2, cmd := update(m, ui.LoginMsg{Username: "bad", Password: "wrong"})
	loginResult := runCmd(cmd)
	m3, _ := update(m2, loginResult)

	if m3.authenticated {
		t.Error("should not be authenticated after failed login")
	}
	if m3.screen != ScreenLogin {
		t.Errorf("should stay on ScreenLogin after failed login, got %v", m3.screen)
	}
}

// ─── Screen navigation ────────────────────────────────────────────────────────

func TestScreenNavigation_ToJobs(t *testing.T) {
	m, _ := newTestModel()
	m = loginModel(m)

	if m.screen != ScreenJobs {
		t.Errorf("expect ScreenJobs after login, got %v", m.screen)
	}
}

func TestWindowSize_Handled(t *testing.T) {
	m, _ := newTestModel()
	m2, _ := update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m2.width != 120 || m2.height != 40 {
		t.Errorf("expected size (120,40), got (%d,%d)", m2.width, m2.height)
	}
}

// ─── Async data loading ───────────────────────────────────────────────────────

func TestJobsLoaded_Success(t *testing.T) {
	m, mock := newTestModel()
	m = loginModel(m)

	jobs, _ := mock.ListJobs()
	m2, _ := update(m, msgJobsLoaded{jobs: jobs})

	if len(m2.jobList) != len(jobs) {
		t.Errorf("want %d jobs, got %d", len(jobs), len(m2.jobList))
	}
}

func TestJobsLoaded_Error(t *testing.T) {
	m, _ := newTestModel()
	m = loginModel(m)

	m2, _ := update(m, msgJobsLoaded{err: errTest("db error")})
	_ = m2 // error is set on the jobs sub-model; just ensure no panic
}

func TestSourcesLoaded_Success(t *testing.T) {
	m, mock := newTestModel()
	m = loginModel(m)

	srcs, _ := mock.ListSources()
	m2, _ := update(m, msgSourcesLoaded{sources: srcs})

	if len(m2.srcList) != len(srcs) {
		t.Errorf("want %d sources, got %d", len(srcs), len(m2.srcList))
	}
}

func TestDestsLoaded_Success(t *testing.T) {
	m, mock := newTestModel()
	m = loginModel(m)

	dests, _ := mock.ListDestinations()
	m2, _ := update(m, msgDestsLoaded{dests: dests})

	if len(m2.dstList) != len(dests) {
		t.Errorf("want %d dests, got %d", len(dests), len(m2.dstList))
	}
}

// ─── Toast notifications ──────────────────────────────────────────────────────

func TestToast_ShowAndExpire(t *testing.T) {
	m, _ := newTestModel()

	m2, _ := update(m, msgShowToast{msg: "hello toast", isErr: false})
	if m2.toast != "hello toast" {
		t.Errorf("toast should be set, got %q", m2.toast)
	}

	m3, _ := update(m2, msgToastExpired{})
	if m3.toast != "" {
		t.Errorf("toast should be cleared on expiry, got %q", m3.toast)
	}
}

func TestToast_ErrorFlag(t *testing.T) {
	m, _ := newTestModel()
	m2, _ := update(m, msgShowToast{msg: "error!", isErr: true})
	if !m2.toastError {
		t.Error("toastError should be true for error toasts")
	}
}

// ─── Delete operations ────────────────────────────────────────────────────────

func TestJobDeleted_Success(t *testing.T) {
	m, _ := newTestModel()
	m = loginModel(m)

	m2, _ := update(m, msgJobDeleted{err: nil})
	// On success, jobs are reloaded (loading=true on sub-model).
	// Just ensure no panic and model is valid.
	_ = m2
}

func TestJobDeleted_Error(t *testing.T) {
	m, _ := newTestModel()
	m = loginModel(m)

	m2, _ := update(m, msgJobDeleted{err: errTest("delete failed")})
	_ = m2
}

// ─── Login state helpers ──────────────────────────────────────────────────────

// loginModel performs a synchronous login by injecting messages directly.
func loginModel(m Model) Model {
	m2, cmd := update(m, ui.LoginMsg{Username: "admin", Password: "pass"})
	if cmd != nil {
		result := runCmd(cmd)
		m2, _ = update(m2, result)
	}
	return m2
}

// ─── Test error type ──────────────────────────────────────────────────────────

type errTest string

func (e errTest) Error() string { return string(e) }
