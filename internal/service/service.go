// Package service provides an HTTP client wrapper for the OLake BFF server.
//
// Architecture note (see docs/05-go-pivot-notes.md):
// The OLake BFF server is the correct integration point. It handles:
//   - Temporal workflow orchestration (discover, check, spec, sync)
//   - Config encryption/decryption (AES-256-GCM or AWS KMS)
//   - Docker image version lookups
//   - PostgreSQL-backed sessions
//
// Direct import of github.com/datazip-inc/olake-ui/server packages was investigated
// but deferred because:
//   1. The server uses Beego framework with global ORM registration (init() side effects)
//   2. Temporal client init panics without TEMPORAL_ADDRESS env var
//   3. Session management is tightly coupled to Beego's HTTP server
//
// Therefore, the TUI talks to the BFF server via HTTP (same as the web frontend).
// When --api-url is provided, all operations go through the BFF REST API.
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const (
	DefaultProjectID = "123"
	DefaultAPIURL    = "http://localhost:8000"
)

// APIResponse is the standard BFF response envelope.
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// Source represents a configured data source connector.
type Source struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	Jobs      []EntityJob `json:"jobs"`
}

// Destination represents a configured data destination connector.
type Destination struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	Jobs      []EntityJob `json:"jobs"`
}

// EntityJob is a job summary attached to a source or destination.
type EntityJob struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	JobName         string `json:"job_name"`
	Activate        bool   `json:"activate"`
	LastRunTime     string `json:"last_run_time"`
	LastRunState    string `json:"last_run_state"`
	SourceName      string `json:"source_name,omitempty"`
	SourceType      string `json:"source_type,omitempty"`
	DestinationName string `json:"destination_name,omitempty"`
	DestinationType string `json:"destination_type,omitempty"`
}

// Job represents a sync job combining a source, destination, and schedule.
type Job struct {
	ID               int             `json:"id"`
	Name             string          `json:"name"`
	Source           JobConnector    `json:"source"`
	Destination      JobConnector    `json:"destination"`
	StreamsConfig    string          `json:"streams_config"`
	Frequency        string          `json:"frequency"`
	LastRunType      string          `json:"last_run_type"`
	LastRunState     string          `json:"last_run_state"`
	LastRunTime      string          `json:"last_run_time"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CreatedBy        string          `json:"created_by"`
	UpdatedBy        string          `json:"updated_by"`
	Activate         bool            `json:"activate"`
	AdvancedSettings *AdvancedSettings `json:"advanced_settings"`
}

// JobConnector is the embedded source/destination info within a Job.
type JobConnector struct {
	ID      int    `json:"id,omitempty"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Config  string `json:"config"`
}

// AdvancedSettings holds optional tuning parameters for a job.
type AdvancedSettings struct {
	MaxDiscoverThreads *int `json:"max_discover_threads"`
}

// JobTask is a historical execution record for a job.
type JobTask struct {
	Runtime   string `json:"runtime"`
	StartTime string `json:"start_time"`
	Status    string `json:"status"`
	FilePath  string `json:"file_path"`
	JobType   string `json:"job_type"`
}

// LogEntry is a single structured log line.
type LogEntry struct {
	Level   string `json:"level"`
	Time    string `json:"time"`
	Message string `json:"message"`
}

// TaskLogsResponse is the paginated log response.
type TaskLogsResponse struct {
	Logs          []LogEntry `json:"logs"`
	OlderCursor   int64      `json:"older_cursor"`
	NewerCursor   int64      `json:"newer_cursor"`
	HasMoreOlder  bool       `json:"has_more_older"`
	HasMoreNewer  bool       `json:"has_more_newer"`
}

// TestConnectionResult is the result of a connection test.
type TestConnectionResult struct {
	ConnectionResult struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"connection_result"`
	Logs []LogEntry `json:"logs"`
}

// SystemSettings holds project-level settings.
type SystemSettings struct {
	ID              int    `json:"id"`
	ProjectID       string `json:"project_id"`
	WebhookAlertURL string `json:"webhook_alert_url"`
}

// EntityBase is the minimal payload for creating a source or destination.
type EntityBase struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Config  string `json:"config"`
}

// Manager is the service layer that communicates with the OLake BFF server.
type Manager struct {
	baseURL   string
	projectID string
	client    *http.Client
	token     string
	username  string
}

// New creates a Manager pointing at the given BFF URL.
func New(apiURL string) *Manager {
	jar, _ := cookiejar.New(nil)
	return &Manager{
		baseURL:   apiURL,
		projectID: DefaultProjectID,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

// projectPath builds a project-scoped API path.
func (m *Manager) projectPath(path string) string {
	return fmt.Sprintf("%s/api/v1/project/%s%s", m.baseURL, m.projectID, path)
}

// do performs an authenticated HTTP request and decodes the response.
func (m *Manager) do(method, url string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — please log in")
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		if errResp.Message != "" {
			return fmt.Errorf("%s", errResp.Message)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

// Login authenticates against the BFF and stores the session token.
func (m *Manager) Login(username, password string) error {
	payload := map[string]string{"username": username, "password": password}
	var resp APIResponse[map[string]string]
	if err := m.do("POST", m.baseURL+"/login", payload, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	m.token = "authenticated"
	m.username = username
	return nil
}

// IsAuthenticated returns true if a token is present.
func (m *Manager) IsAuthenticated() bool {
	return m.token != ""
}

// Username returns the logged-in username.
func (m *Manager) Username() string {
	return m.username
}

// CheckAuth verifies the current session is still valid.
func (m *Manager) CheckAuth() error {
	var resp APIResponse[map[string]string]
	return m.do("GET", m.baseURL+"/auth/check", nil, &resp)
}

// --- Sources ---

// ListSources returns all sources for the project.
func (m *Manager) ListSources() ([]Source, error) {
	var resp APIResponse[[]Source]
	if err := m.do("GET", m.projectPath("/sources"), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetSource returns a single source by ID.
func (m *Manager) GetSource(id int) (*Source, error) {
	var resp APIResponse[Source]
	if err := m.do("GET", fmt.Sprintf("%s/%d", m.projectPath("/sources"), id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateSource creates a new source.
func (m *Manager) CreateSource(s EntityBase) (*EntityBase, error) {
	var resp APIResponse[EntityBase]
	if err := m.do("POST", m.projectPath("/sources"), s, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateSource updates an existing source.
func (m *Manager) UpdateSource(id int, s EntityBase) (*Source, error) {
	var resp APIResponse[Source]
	if err := m.do("PUT", fmt.Sprintf("%s/%d", m.projectPath("/sources"), id), s, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteSource deletes a source by ID.
func (m *Manager) DeleteSource(id int) error {
	return m.do("DELETE", fmt.Sprintf("%s/%d", m.projectPath("/sources"), id), nil, nil)
}

// TestSource tests a source connection.
func (m *Manager) TestSource(s EntityBase) (*TestConnectionResult, error) {
	var resp APIResponse[TestConnectionResult]
	payload := map[string]string{
		"type":    s.Type,
		"version": s.Version,
		"config":  s.Config,
	}
	client := &http.Client{Jar: m.client.Jar} // no timeout for test
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", m.projectPath("/sources/test"), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if m.token != "" {
		req.Header.Set("Authorization", "Bearer "+m.token)
	}
	httpResp, err := client.Do(req)
	if err != nil {
		return &TestConnectionResult{}, err
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(httpResp.Body)
	_ = json.Unmarshal(body, &resp)
	return &resp.Data, nil
}

// --- Destinations ---

// ListDestinations returns all destinations for the project.
func (m *Manager) ListDestinations() ([]Destination, error) {
	var resp APIResponse[[]Destination]
	if err := m.do("GET", m.projectPath("/destinations"), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetDestination returns a single destination by ID.
func (m *Manager) GetDestination(id int) (*Destination, error) {
	var resp APIResponse[Destination]
	if err := m.do("GET", fmt.Sprintf("%s/%d", m.projectPath("/destinations"), id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateDestination creates a new destination.
func (m *Manager) CreateDestination(d EntityBase) (*EntityBase, error) {
	var resp APIResponse[EntityBase]
	if err := m.do("POST", m.projectPath("/destinations"), d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateDestination updates an existing destination.
func (m *Manager) UpdateDestination(id int, d EntityBase) (*EntityBase, error) {
	var resp APIResponse[EntityBase]
	if err := m.do("PUT", fmt.Sprintf("%s/%d", m.projectPath("/destinations"), id), d, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteDestination deletes a destination by ID.
func (m *Manager) DeleteDestination(id int) error {
	return m.do("DELETE", fmt.Sprintf("%s/%d", m.projectPath("/destinations"), id), nil, nil)
}

// --- Jobs ---

// ListJobs returns all jobs for the project.
func (m *Manager) ListJobs() ([]Job, error) {
	var resp APIResponse[[]Job]
	if err := m.do("GET", m.projectPath("/jobs"), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetJob returns a single job by ID.
func (m *Manager) GetJob(id int) (*Job, error) {
	var resp APIResponse[Job]
	if err := m.do("GET", fmt.Sprintf("%s/%d", m.projectPath("/jobs"), id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteJob deletes a job by ID.
func (m *Manager) DeleteJob(id int) error {
	return m.do("DELETE", fmt.Sprintf("%s/%d", m.projectPath("/jobs"), id), nil, nil)
}

// TriggerSync triggers an immediate sync for a job.
func (m *Manager) TriggerSync(id int) error {
	return m.do("POST", fmt.Sprintf("%s/%d/sync", m.projectPath("/jobs"), id), map[string]any{}, nil)
}

// CancelJob cancels a running job.
func (m *Manager) CancelJob(id int) error {
	return m.do("GET", fmt.Sprintf("%s/%d/cancel", m.projectPath("/jobs"), id), nil, nil)
}

// ActivateJob pauses or resumes a job.
func (m *Manager) ActivateJob(id int, activate bool) error {
	return m.do("POST", fmt.Sprintf("%s/%d/activate", m.projectPath("/jobs"), id),
		map[string]bool{"activate": activate}, nil)
}

// ListJobTasks returns the run history for a job.
func (m *Manager) ListJobTasks(jobID int) ([]JobTask, error) {
	var resp APIResponse[[]JobTask]
	if err := m.do("GET", fmt.Sprintf("%s/%d/tasks", m.projectPath("/jobs"), jobID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetTaskLogs returns paginated logs for a job task.
func (m *Manager) GetTaskLogs(jobID int, taskID string, filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error) {
	url := fmt.Sprintf("%s/%d/tasks/%s/logs?cursor=%d&limit=%d&direction=%s",
		m.projectPath("/jobs"), jobID, taskID, cursor, limit, direction)
	payload := map[string]string{"file_path": filePath}
	var resp APIResponse[TaskLogsResponse]
	if err := m.do("POST", url, payload, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetSettings returns project-level settings.
func (m *Manager) GetSettings() (*SystemSettings, error) {
	var resp APIResponse[SystemSettings]
	if err := m.do("GET", m.projectPath("/settings"), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdateSettings saves project-level settings.
func (m *Manager) UpdateSettings(s SystemSettings) error {
	return m.do("PUT", m.projectPath("/settings"), s, nil)
}
