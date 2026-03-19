// Package service provides a mock implementation of the Service interface
// for use in unit and integration tests. It is safe to use without any
// external dependencies (no PostgreSQL, no Temporal).
package service

import (
	"fmt"
	"sync"
	"time"
)

// MockService implements Service with pre-canned responses and configurable
// error injection. It is the primary testing double for all UI tests.
//
// Usage:
//
//	m := NewMockService()
//	m.Sources = []Source{{ID: 1, Name: "test-pg"}}
//	m.LoginErr = fmt.Errorf("bad credentials")
type MockService struct {
	mu sync.Mutex

	// ── Auth state ────────────────────────────────────────────────────────
	authenticated bool
	username      string

	// ── Pre-canned data ───────────────────────────────────────────────────
	Sources      []Source
	Destinations []Destination
	Jobs         []Job
	Tasks        []JobTask
	Settings     *SystemSettings
	Streams      []StreamInfo
	TestResult   *TestConnectionResult

	// ── Pre-canned flags ──────────────────────────────────────────────────
	ClearDestRunning bool

	// ── Error injection ───────────────────────────────────────────────────
	LoginErr           error
	ListSourcesErr     error
	GetSourceErr       error
	CreateSourceErr    error
	UpdateSourceErr    error
	DeleteSourceErr    error
	TestSourceErr      error
	TestDestinationErr error
	DiscoverErr        error
	ListDestsErr       error
	GetDestErr         error
	CreateDestErr      error
	UpdateDestErr      error
	DeleteDestErr      error
	ListJobsErr        error
	GetJobErr          error
	CreateJobErr       error
	UpdateJobMetaErr   error
	DeleteJobErr       error
	TriggerSyncErr     error
	CancelJobErr       error
	ActivateJobErr     error
	ListTasksErr       error
	GetTaskLogsErr     error
	ClearDestErr       error
	GetSettingsErr          error
	UpdateSettingsErr       error
	ValidateSchemaErr       error
	IsNameUniqueErr         error
	GetClearDestStatusErr   error
	RecoverFromClearDestErr error
	UpdateJobFullErr        error

	// ── Call counters (useful for assertions) ─────────────────────────────
	Calls map[string]int
}

// NewMockService returns a MockService populated with sensible defaults.
func NewMockService() *MockService {
	m := &MockService{
		Calls: make(map[string]int),
		Settings: &SystemSettings{
			ID:        1,
			ProjectID: "test-project",
		},
		TestResult: &TestConnectionResult{},
	}
	m.TestResult.ConnectionResult.Status = "success"
	m.TestResult.ConnectionResult.Message = "Connected"
	return m
}

func (m *MockService) record(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls[method]++
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func (m *MockService) Login(username, password string) error {
	m.record("Login")
	if m.LoginErr != nil {
		return m.LoginErr
	}
	m.mu.Lock()
	m.authenticated = true
	m.username = username
	m.mu.Unlock()
	return nil
}

func (m *MockService) IsAuthenticated() bool {
	m.record("IsAuthenticated")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.authenticated
}

func (m *MockService) Username() string {
	m.record("Username")
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.username
}

// ── Sources ───────────────────────────────────────────────────────────────────

func (m *MockService) ListSources() ([]Source, error) {
	m.record("ListSources")
	if m.ListSourcesErr != nil {
		return nil, m.ListSourcesErr
	}
	return m.Sources, nil
}

func (m *MockService) GetSource(id int) (*Source, error) {
	m.record("GetSource")
	if m.GetSourceErr != nil {
		return nil, m.GetSourceErr
	}
	for i := range m.Sources {
		if m.Sources[i].ID == id {
			return &m.Sources[i], nil
		}
	}
	return nil, fmt.Errorf("source not found id[%d]", id)
}

func (m *MockService) CreateSource(s EntityBase) (*EntityBase, error) {
	m.record("CreateSource")
	if m.CreateSourceErr != nil {
		return nil, m.CreateSourceErr
	}
	m.mu.Lock()
	m.Sources = append(m.Sources, Source{
		ID:        len(m.Sources) + 1,
		Name:      s.Name,
		Type:      s.Type,
		Version:   s.Version,
		Config:    s.Config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	m.mu.Unlock()
	return &s, nil
}

func (m *MockService) UpdateSource(id int, s EntityBase) (*Source, error) {
	m.record("UpdateSource")
	if m.UpdateSourceErr != nil {
		return nil, m.UpdateSourceErr
	}
	m.mu.Lock()
	for i := range m.Sources {
		if m.Sources[i].ID == id {
			m.Sources[i].Name = s.Name
			m.Sources[i].Type = s.Type
			m.Sources[i].Version = s.Version
			m.Sources[i].Config = s.Config
			src := m.Sources[i]
			m.mu.Unlock()
			return &src, nil
		}
	}
	m.mu.Unlock()
	return nil, fmt.Errorf("source not found id[%d]", id)
}

func (m *MockService) DeleteSource(id int) error {
	m.record("DeleteSource")
	if m.DeleteSourceErr != nil {
		return m.DeleteSourceErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Sources {
		if m.Sources[i].ID == id {
			m.Sources = append(m.Sources[:i], m.Sources[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("source not found id[%d]", id)
}

func (m *MockService) TestSource(s EntityBase) (*TestConnectionResult, error) {
	m.record("TestSource")
	if m.TestSourceErr != nil {
		return nil, m.TestSourceErr
	}
	return m.TestResult, nil
}

func (m *MockService) TestDestination(d EntityBase, sourceType, sourceVersion string) (*TestConnectionResult, error) {
	m.record("TestDestination")
	if m.TestDestinationErr != nil {
		return nil, m.TestDestinationErr
	}
	return m.TestResult, nil
}

func (m *MockService) DiscoverStreams(sourceID int) ([]StreamInfo, error) {
	m.record("DiscoverStreams")
	if m.DiscoverErr != nil {
		return nil, m.DiscoverErr
	}
	return m.Streams, nil
}

// ── Destinations ─────────────────────────────────────────────────────────────

func (m *MockService) ListDestinations() ([]Destination, error) {
	m.record("ListDestinations")
	if m.ListDestsErr != nil {
		return nil, m.ListDestsErr
	}
	return m.Destinations, nil
}

func (m *MockService) GetDestination(id int) (*Destination, error) {
	m.record("GetDestination")
	if m.GetDestErr != nil {
		return nil, m.GetDestErr
	}
	for i := range m.Destinations {
		if m.Destinations[i].ID == id {
			return &m.Destinations[i], nil
		}
	}
	return nil, fmt.Errorf("destination not found id[%d]", id)
}

func (m *MockService) CreateDestination(d EntityBase) (*EntityBase, error) {
	m.record("CreateDestination")
	if m.CreateDestErr != nil {
		return nil, m.CreateDestErr
	}
	m.mu.Lock()
	m.Destinations = append(m.Destinations, Destination{
		ID:        len(m.Destinations) + 1,
		Name:      d.Name,
		Type:      d.Type,
		Version:   d.Version,
		Config:    d.Config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	m.mu.Unlock()
	return &d, nil
}

func (m *MockService) UpdateDestination(id int, d EntityBase) (*EntityBase, error) {
	m.record("UpdateDestination")
	if m.UpdateDestErr != nil {
		return nil, m.UpdateDestErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Destinations {
		if m.Destinations[i].ID == id {
			m.Destinations[i].Name = d.Name
			m.Destinations[i].Type = d.Type
			m.Destinations[i].Version = d.Version
			m.Destinations[i].Config = d.Config
			return &d, nil
		}
	}
	return nil, fmt.Errorf("destination not found id[%d]", id)
}

func (m *MockService) DeleteDestination(id int) error {
	m.record("DeleteDestination")
	if m.DeleteDestErr != nil {
		return m.DeleteDestErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Destinations {
		if m.Destinations[i].ID == id {
			m.Destinations = append(m.Destinations[:i], m.Destinations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("destination not found id[%d]", id)
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

func (m *MockService) ListJobs() ([]Job, error) {
	m.record("ListJobs")
	if m.ListJobsErr != nil {
		return nil, m.ListJobsErr
	}
	return m.Jobs, nil
}

func (m *MockService) GetJob(id int) (*Job, error) {
	m.record("GetJob")
	if m.GetJobErr != nil {
		return nil, m.GetJobErr
	}
	for i := range m.Jobs {
		if m.Jobs[i].ID == id {
			return &m.Jobs[i], nil
		}
	}
	return nil, fmt.Errorf("job not found id[%d]", id)
}

func (m *MockService) CreateJob(name string, sourceID, destID int, frequency string, streams []StreamConfig) (*Job, error) {
	m.record("CreateJob")
	if m.CreateJobErr != nil {
		return nil, m.CreateJobErr
	}
	j := Job{
		ID:        len(m.Jobs) + 1,
		Name:      name,
		Frequency: frequency,
		Activate:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// find source/dest by ID
	for _, s := range m.Sources {
		if s.ID == sourceID {
			j.Source = JobConnector{ID: s.ID, Name: s.Name, Type: s.Type}
		}
	}
	for _, d := range m.Destinations {
		if d.ID == destID {
			j.Destination = JobConnector{ID: d.ID, Name: d.Name, Type: d.Type}
		}
	}
	m.mu.Lock()
	m.Jobs = append(m.Jobs, j)
	m.mu.Unlock()
	return &j, nil
}

func (m *MockService) UpdateJobMeta(id int, name, frequency string) error {
	m.record("UpdateJobMeta")
	if m.UpdateJobMetaErr != nil {
		return m.UpdateJobMetaErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Jobs {
		if m.Jobs[i].ID == id {
			m.Jobs[i].Name = name
			m.Jobs[i].Frequency = frequency
			return nil
		}
	}
	return fmt.Errorf("job not found id[%d]", id)
}

func (m *MockService) DeleteJob(id int) error {
	m.record("DeleteJob")
	if m.DeleteJobErr != nil {
		return m.DeleteJobErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Jobs {
		if m.Jobs[i].ID == id {
			m.Jobs = append(m.Jobs[:i], m.Jobs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("job not found id[%d]", id)
}

func (m *MockService) TriggerSync(id int) error {
	m.record("TriggerSync")
	return m.TriggerSyncErr
}

func (m *MockService) CancelJob(id int) error {
	m.record("CancelJob")
	return m.CancelJobErr
}

func (m *MockService) ActivateJob(id int, activate bool) error {
	m.record("ActivateJob")
	if m.ActivateJobErr != nil {
		return m.ActivateJobErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Jobs {
		if m.Jobs[i].ID == id {
			m.Jobs[i].Activate = activate
			return nil
		}
	}
	return fmt.Errorf("job not found id[%d]", id)
}

func (m *MockService) ListJobTasks(jobID int) ([]JobTask, error) {
	m.record("ListJobTasks")
	if m.ListTasksErr != nil {
		return nil, m.ListTasksErr
	}
	return m.Tasks, nil
}

func (m *MockService) GetTaskLogs(jobID int, taskID string, filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error) {
	m.record("GetTaskLogs")
	if m.GetTaskLogsErr != nil {
		return nil, m.GetTaskLogsErr
	}
	return &TaskLogsResponse{
		Logs: []LogEntry{{Level: "info", Time: time.Now().Format(time.RFC3339), Message: "mock log entry"}},
	}, nil
}

func (m *MockService) ClearDestination(id int) error {
	m.record("ClearDestination")
	return m.ClearDestErr
}

// ── Settings ─────────────────────────────────────────────────────────────────

func (m *MockService) GetSettings() (*SystemSettings, error) {
	m.record("GetSettings")
	if m.GetSettingsErr != nil {
		return nil, m.GetSettingsErr
	}
	if m.Settings == nil {
		return &SystemSettings{ProjectID: "test-project"}, nil
	}
	return m.Settings, nil
}

func (m *MockService) UpdateSettings(s SystemSettings) error {
	m.record("UpdateSettings")
	if m.UpdateSettingsErr != nil {
		return m.UpdateSettingsErr
	}
	m.mu.Lock()
	m.Settings = &s
	m.mu.Unlock()
	return nil
}

// ── Schema ────────────────────────────────────────────────────────────────────

func (m *MockService) ValidateSchema() error {
	m.record("ValidateSchema")
	return m.ValidateSchemaErr
}

func (m *MockService) GetCompatibleVersion() string {
	return "olake >= 1.0.0 (mock)"
}

// ── Name Uniqueness ───────────────────────────────────────────────────────────

func (m *MockService) IsNameUnique(entityType string, name string) (bool, error) {
	m.record("IsNameUnique")
	if m.IsNameUniqueErr != nil {
		return false, m.IsNameUniqueErr
	}
	return true, nil
}

// ── Clear Destination Status ──────────────────────────────────────────────────

func (m *MockService) GetClearDestStatus(jobID int) (bool, error) {
	m.record("GetClearDestStatus")
	if m.GetClearDestStatusErr != nil {
		return false, m.GetClearDestStatusErr
	}
	return m.ClearDestRunning, nil
}

func (m *MockService) RecoverFromClearDest(jobID int) error {
	m.record("RecoverFromClearDest")
	return m.RecoverFromClearDestErr
}

// ── Full Job Update ───────────────────────────────────────────────────────────

func (m *MockService) UpdateJobFull(id int, name string, sourceID, destID int, frequency string, streams []StreamConfig, activate bool, advancedSettings *AdvancedSettings) error {
	m.record("UpdateJobFull")
	if m.UpdateJobFullErr != nil {
		return m.UpdateJobFullErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.Jobs {
		if m.Jobs[i].ID == id {
			m.Jobs[i].Name = name
			m.Jobs[i].Frequency = frequency
			m.Jobs[i].Activate = activate
			return nil
		}
	}
	return fmt.Errorf("job not found id[%d]", id)
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

func (m *MockService) Close() {
	m.record("Close")
}

// compile-time check
var _ Service = (*MockService)(nil)
