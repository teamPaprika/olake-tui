// Package service provides the Service interface that the UI layer depends on.
//
// By depending on the interface rather than *Manager, the UI can be tested
// with mock implementations and remain decoupled from transport decisions.
package service

// Service is the contract the UI layer (internal/app) uses.
// It is intentionally defined in terms of the domain types already declared in
// this package (Source, Destination, Job, etc.) so that no import cycle is
// introduced.
//
// The concrete implementation is *Manager (service.go).  All methods listed
// here must have matching signatures on Manager; compile-time enforcement is
// done via the var _ Service = (*Manager)(nil) assertion at the bottom.
type Service interface {
	// ── Auth ─────────────────────────────────────────────────────────────

	// Login authenticates the user against the PostgreSQL user table.
	Login(username, password string) error

	// IsAuthenticated reports whether a successful Login has been called.
	IsAuthenticated() bool

	// Username returns the currently authenticated user's name.
	Username() string

	// ── Sources ───────────────────────────────────────────────────────────

	ListSources() ([]Source, error)
	GetSource(id int) (*Source, error)
	CreateSource(s EntityBase) (*EntityBase, error)
	UpdateSource(id int, s EntityBase) (*Source, error)
	DeleteSource(id int) error
	TestSource(s EntityBase) (*TestConnectionResult, error)
	DiscoverStreams(sourceID int) ([]StreamInfo, error)

	// ── Destinations ──────────────────────────────────────────────────────

	ListDestinations() ([]Destination, error)
	GetDestination(id int) (*Destination, error)
	CreateDestination(d EntityBase) (*EntityBase, error)
	UpdateDestination(id int, d EntityBase) (*EntityBase, error)
	DeleteDestination(id int) error

	// ── Jobs ──────────────────────────────────────────────────────────────

	ListJobs() ([]Job, error)
	GetJob(id int) (*Job, error)
	CreateJob(name string, sourceID, destID int, frequency string, streams []StreamConfig) (*Job, error)
	UpdateJobMeta(id int, name, frequency string) error
	DeleteJob(id int) error
	TriggerSync(id int) error
	CancelJob(id int) error
	ActivateJob(id int, activate bool) error
	ListJobTasks(jobID int) ([]JobTask, error)
	GetTaskLogs(jobID int, taskID string, filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error)
	ClearDestination(id int) error

	// ── Settings ──────────────────────────────────────────────────────────

	GetSettings() (*SystemSettings, error)
	UpdateSettings(s SystemSettings) error

	// ── Schema ────────────────────────────────────────────────────────────

	// ValidateSchema checks that the connected database has the expected
	// OLake schema and returns an error with migration guidance if not.
	ValidateSchema() error

	// GetCompatibleVersion returns the minimum OLake version this TUI build
	// is compatible with (e.g. "olake >= 1.0.0").
	GetCompatibleVersion() string

	// ── Lifecycle ─────────────────────────────────────────────────────────

	// Close releases the database connection and Temporal client.
	Close()
}

// Compile-time assertion: *Manager must satisfy Service.
// If any method is missing or has the wrong signature this line will fail
// with a clear compiler error pointing to the offending method.
var _ Service = (*Manager)(nil)
