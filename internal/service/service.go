// Package service provides a direct database + Temporal client for the OLake TUI.
//
// Architecture (see docs/05-go-pivot-notes.md):
//
// Direct import of github.com/datazip-inc/olake-ui/server packages is not feasible
// because the BFF server uses the Beego framework, whose constants.Init() calls
// checkForRequiredVariables() which panics at import time if OLAKE_POSTGRES_* env vars
// are not set. The BFF also uses web.AppConfig throughout its service layer.
//
// This package is a "forked service layer":
//   - Uses standard database/sql (lib/pq driver) instead of Beego ORM
//   - Uses go.temporal.io/sdk/client directly
//   - Implements the same AES-256-GCM encryption/decryption used by the BFF
//   - Matches the same DB schema and table-naming convention
//   - No Beego dependency whatsoever
//
// DB table names match the BFF's "dev" run mode: olake-dev-<entity>.
// Pass --run-mode=<mode> to override (dev/prod/staging).
package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultProjectID    = "123"
	DefaultTemporalHost = "localhost:7233"
	DefaultRunMode      = "dev"

	// Temporal task queue used by the olake worker.
	TemporalTaskQueue = "OLAKE_DOCKER_TASK_QUEUE"

	// Temporal workflow type names (must match worker registration).
	WorkflowTypeExecute = "ExecuteWorkflow"
	WorkflowTypeRunSync = "RunSyncWorkflow"

	// DefaultConfigDir is the shared volume path between BFF/TUI and the Temporal worker.
	DefaultConfigDir = "/tmp/olake-config"

	// Timeout constants for Temporal workflows.
	discoverTimeout       = 10 * time.Minute
	checkTimeout          = 10 * time.Minute
	syncTimeout           = 30 * 24 * time.Hour
	clearDestTimeout      = 30 * 24 * time.Hour
	cancelSyncWaitTimeout = 30 * time.Second
)

// ─── Temporal execution request (mirrors BFF's ExecutionRequest) ─────────────

// temporalCommand is the CLI command dispatched to the worker.
type temporalCommand string

const (
	cmdDiscover         temporalCommand = "discover"
	cmdCheck            temporalCommand = "check"
	cmdSync             temporalCommand = "sync"
	cmdClearDestination temporalCommand = "clear-destination"
)

// jobConfig is a named file that the worker should write to the shared volume.
type jobConfig struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// executionRequest is the payload sent to the Temporal worker for all
// one-shot (non-scheduled) workflow executions.  Field names must match
// the worker's own ExecutionRequest struct.
type executionRequest struct {
	Command       temporalCommand `json:"command"`
	ConnectorType string          `json:"connector_type"`
	Version       string          `json:"version"`
	Args          []string        `json:"args"`
	Configs       []jobConfig     `json:"configs"`
	WorkflowID    string          `json:"workflow_id"`
	ProjectID     string          `json:"project_id"`
	JobID         int             `json:"job_id"`
	Timeout       time.Duration   `json:"timeout"`
	OutputFile    string          `json:"output_file"`
	TempPath      string          `json:"temp_path,omitempty"`
}

// ─── Domain types (kept identical to the HTTP version) ─────────────────────

// Source represents a configured data source connector.
type Source struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Version   string      `json:"version"`
	Config    string      `json:"config"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	CreatedBy string      `json:"created_by"`
	UpdatedBy string      `json:"updated_by"`
	Jobs      []EntityJob `json:"jobs"`
	JobCount  int         `json:"job_count"`
}

// Destination represents a configured data destination connector.
type Destination struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Version   string      `json:"version"`
	Config    string      `json:"config"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	CreatedBy string      `json:"created_by"`
	UpdatedBy string      `json:"updated_by"`
	Jobs      []EntityJob `json:"jobs"`
	JobCount  int         `json:"job_count"`
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
	ID               int               `json:"id"`
	Name             string            `json:"name"`
	Source           JobConnector      `json:"source"`
	Destination      JobConnector      `json:"destination"`
	StreamsConfig    string            `json:"streams_config"`
	Frequency        string            `json:"frequency"`
	LastRunType      string            `json:"last_run_type"`
	LastRunState     string            `json:"last_run_state"`
	LastRunTime      string            `json:"last_run_time"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CreatedBy        string            `json:"created_by"`
	UpdatedBy        string            `json:"updated_by"`
	Activate         bool              `json:"activate"`
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
	Logs         []LogEntry `json:"logs"`
	OlderCursor  int64      `json:"older_cursor"`
	NewerCursor  int64      `json:"newer_cursor"`
	HasMoreOlder bool       `json:"has_more_older"`
	HasMoreNewer bool       `json:"has_more_newer"`
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

// ─── Manager ────────────────────────────────────────────────────────────────

// Manager is the TUI service layer that talks directly to PostgreSQL and Temporal.
type Manager struct {
	db            *sql.DB
	temporal      client.Client
	projectID     string
	runMode       string
	encryptionKey string // raw key (env var value); empty = no encryption

	// authMu protects the auth fields below, which may be written from goroutines.
	authMu        sync.RWMutex
	username      string
	userID        int
	authenticated bool
}

// Config holds initialization parameters.
type Config struct {
	DBURL         string // postgres connection string
	TemporalHost  string // e.g. localhost:7233
	ProjectID     string
	RunMode       string // dev | prod | staging
	EncryptionKey string // OLAKE_SECRET_KEY value (optional)
}

// New connects to PostgreSQL and Temporal, returning a ready Manager.
// Temporal connection is attempted but failure is non-fatal (features requiring
// Temporal will return errors at call time).
func New(cfg Config) (*Manager, error) {
	if cfg.ProjectID == "" {
		cfg.ProjectID = DefaultProjectID
	}
	if cfg.RunMode == "" {
		cfg.RunMode = DefaultRunMode
	}
	if cfg.TemporalHost == "" {
		cfg.TemporalHost = DefaultTemporalHost
	}

	// Validate runMode to prevent SQL injection via table names.
	switch cfg.RunMode {
	case "dev", "prod", "staging":
		// ok
	default:
		return nil, fmt.Errorf("invalid run mode %q: must be dev, prod, or staging", cfg.RunMode)
	}

	if cfg.DBURL == "" {
		return nil, fmt.Errorf("OLAKE_DB_URL is required (PostgreSQL connection string)")
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	m := &Manager{
		db:            db,
		projectID:     cfg.ProjectID,
		runMode:       cfg.RunMode,
		encryptionKey: cfg.EncryptionKey,
	}

	// Connect Temporal (best-effort)
	tc, err := client.Dial(client.Options{HostPort: cfg.TemporalHost})
	if err == nil {
		m.temporal = tc
	}
	// If Temporal dial fails we continue — features that need it will fail at call time.

	return m, nil
}

// Close releases DB and Temporal resources.
func (m *Manager) Close() {
	if m.db != nil {
		_ = m.db.Close()
	}
	if m.temporal != nil {
		m.temporal.Close()
	}
}

// ─── Table names (matches BFF convention: olake-<runMode>-<entity>) ─────────

func (m *Manager) tbl(entity string) string {
	return fmt.Sprintf(`"olake-%s-%s"`, m.runMode, entity)
}

// ─── Encryption (same AES-256-GCM as BFF utils/encryption.go) ───────────────

func (m *Manager) encryptionKeyBytes() []byte {
	if strings.TrimSpace(m.encryptionKey) == "" {
		return nil
	}
	h := sha256.Sum256([]byte(m.encryptionKey))
	return h[:]
}

func (m *Manager) encrypt(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return plaintext, nil
	}
	key := m.encryptionKeyBytes()
	if key == nil {
		return plaintext, nil // no encryption configured
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	// BFF stores as JSON-quoted base64 string
	b64 := base64.StdEncoding.EncodeToString(ct)
	out, err := json.Marshal(b64)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m *Manager) decrypt(encryptedText string) (string, error) {
	if strings.TrimSpace(encryptedText) == "" {
		return encryptedText, nil
	}
	key := m.encryptionKeyBytes()
	if key == nil {
		return encryptedText, nil // no encryption configured
	}

	var b64 string
	if err := json.Unmarshal([]byte(encryptedText), &b64); err != nil {
		// Failed to unmarshal as JSON-quoted string: data is either corrupted or
		// was stored without encryption. Return an error rather than silently
		// handing back raw ciphertext, which would appear as garbage config.
		return "", fmt.Errorf("decrypt: failed to parse encrypted data (corrupted or wrong key): %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// ─── Auth ────────────────────────────────────────────────────────────────────

// Login authenticates against the PostgreSQL user table.
func (m *Manager) Login(username, password string) error {
	query := fmt.Sprintf(`SELECT id, password FROM %s WHERE username = $1 LIMIT 1`, m.tbl("user"))
	var id int
	var hashedPwd string
	err := m.db.QueryRowContext(context.Background(), query, username).Scan(&id, &hashedPwd)
	if err == sql.ErrNoRows {
		return fmt.Errorf("invalid credentials")
	}
	if err != nil {
		return fmt.Errorf("login query: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(password)); err != nil {
		return fmt.Errorf("invalid credentials")
	}
	m.authMu.Lock()
	m.authenticated = true
	m.username = username
	m.userID = id
	m.authMu.Unlock()
	return nil
}

// IsAuthenticated returns true after a successful Login call.
func (m *Manager) IsAuthenticated() bool {
	m.authMu.RLock()
	defer m.authMu.RUnlock()
	return m.authenticated
}

// Username returns the logged-in username.
func (m *Manager) Username() string {
	m.authMu.RLock()
	defer m.authMu.RUnlock()
	return m.username
}

// CheckAuth is a no-op for the direct service layer (always returns nil when authenticated).
func (m *Manager) CheckAuth() error {
	m.authMu.RLock()
	defer m.authMu.RUnlock()
	if !m.authenticated {
		return fmt.Errorf("not authenticated")
	}
	return nil
}

// currentUserID returns the authenticated user's ID safely.
func (m *Manager) currentUserID() int {
	m.authMu.RLock()
	defer m.authMu.RUnlock()
	return m.userID
}

// ─── Sources ─────────────────────────────────────────────────────────────────

// ListSources returns all sources for the project, including associated job counts.
func (m *Manager) ListSources() ([]Source, error) {
	q := fmt.Sprintf(`
		SELECT s.id, s.name, s.type, s.version, s.config, s.created_at, s.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by,
		       (SELECT COUNT(*) FROM %s j WHERE j.source_id = s.id AND j.deleted_at IS NULL) AS job_count
		FROM %s s
		LEFT JOIN %s cu ON s.created_by_id = cu.id
		LEFT JOIN %s uu ON s.updated_by_id = uu.id
		WHERE s.project_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.updated_at DESC`,
		m.tbl("job"), m.tbl("source"), m.tbl("user"), m.tbl("user"))

	rows, err := m.db.QueryContext(context.Background(), q, m.projectID)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		var encCfg string
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Version, &encCfg,
			&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy, &s.JobCount); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		s.Config, err = m.decrypt(encCfg)
		if err != nil {
			return nil, fmt.Errorf("decrypt source config id[%d]: %w", s.ID, err)
		}
		sources = append(sources, s)
	}
	if sources == nil {
		sources = []Source{}
	}
	return sources, rows.Err()
}

// GetSource returns a single source by ID.
func (m *Manager) GetSource(id int) (*Source, error) {
	q := fmt.Sprintf(`
		SELECT s.id, s.name, s.type, s.version, s.config, s.created_at, s.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by
		FROM %s s
		LEFT JOIN %s cu ON s.created_by_id = cu.id
		LEFT JOIN %s uu ON s.updated_by_id = uu.id
		WHERE s.id = $1 AND s.deleted_at IS NULL`,
		m.tbl("source"), m.tbl("user"), m.tbl("user"))

	var s Source
	var encCfg string
	err := m.db.QueryRowContext(context.Background(), q, id).Scan(
		&s.ID, &s.Name, &s.Type, &s.Version, &encCfg,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("source not found id[%d]", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get source: %w", err)
	}
	s.Config, err = m.decrypt(encCfg)
	if err != nil {
		return nil, fmt.Errorf("decrypt source config: %w", err)
	}
	return &s, nil
}

// CreateSource creates a new source.
func (m *Manager) CreateSource(s EntityBase) (*EntityBase, error) {
	unique, err := m.IsNameUnique("source", s.Name)
	if err != nil {
		return nil, fmt.Errorf("check source name uniqueness: %w", err)
	}
	if !unique {
		return nil, fmt.Errorf("source name %q already exists in this project", s.Name)
	}
	encCfg, err := m.encrypt(s.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt source config: %w", err)
	}
	q := fmt.Sprintf(`
		INSERT INTO %s (name, type, version, config, project_id, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6, NOW(), NOW())`,
		m.tbl("source"))
	_, err = m.db.ExecContext(context.Background(), q, s.Name, s.Type, s.Version, encCfg, m.projectID, m.currentUserID())
	if err != nil {
		return nil, fmt.Errorf("create source: %w", err)
	}
	return &s, nil
}

// UpdateSource updates an existing source.
func (m *Manager) UpdateSource(id int, s EntityBase) (*Source, error) {
	encCfg, err := m.encrypt(s.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt source config: %w", err)
	}
	q := fmt.Sprintf(`
		UPDATE %s SET name=$1, type=$2, version=$3, config=$4, updated_by_id=$5, updated_at=NOW()
		WHERE id=$6`,
		m.tbl("source"))
	_, err = m.db.ExecContext(context.Background(), q, s.Name, s.Type, s.Version, encCfg, m.currentUserID(), id)
	if err != nil {
		return nil, fmt.Errorf("update source: %w", err)
	}
	return m.GetSource(id)
}

// DeleteSource soft-deletes a source by ID (sets deleted_at = NOW()).
// This matches the BFF's Beego ORM soft-delete behavior so that audit trails
// are preserved and BFF users can still see historical data.
func (m *Manager) DeleteSource(id int) error {
	// Check no jobs reference this source
	count, err := m.countJobsBySource(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("source is used by %d job(s); delete those jobs first", count)
	}
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=NOW(), updated_at=NOW(), updated_by_id=$2 WHERE id=$1 AND deleted_at IS NULL`, m.tbl("source"))
	uid := m.currentUserID()
	_, err = m.db.ExecContext(context.Background(), q, id, uid)
	return err
}

func (m *Manager) countJobsBySource(sourceID int) (int, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE source_id=$1 AND deleted_at IS NULL`, m.tbl("job"))
	var n int
	err := m.db.QueryRowContext(context.Background(), q, sourceID).Scan(&n)
	return n, err
}

// ─── Destinations ─────────────────────────────────────────────────────────────

// ListDestinations returns all destinations for the project, including associated job counts.
func (m *Manager) ListDestinations() ([]Destination, error) {
	q := fmt.Sprintf(`
		SELECT d.id, d.name, d.dest_type, d.version, d.config, d.created_at, d.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by,
		       (SELECT COUNT(*) FROM %s j WHERE j.dest_id = d.id AND j.deleted_at IS NULL) AS job_count
		FROM %s d
		LEFT JOIN %s cu ON d.created_by_id = cu.id
		LEFT JOIN %s uu ON d.updated_by_id = uu.id
		WHERE d.project_id = $1 AND d.deleted_at IS NULL
		ORDER BY d.updated_at DESC`,
		m.tbl("job"), m.tbl("destination"), m.tbl("user"), m.tbl("user"))

	rows, err := m.db.QueryContext(context.Background(), q, m.projectID)
	if err != nil {
		return nil, fmt.Errorf("list destinations: %w", err)
	}
	defer rows.Close()

	var dests []Destination
	for rows.Next() {
		var d Destination
		var encCfg string
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Version, &encCfg,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.JobCount); err != nil {
			return nil, fmt.Errorf("scan destination: %w", err)
		}
		d.Config, err = m.decrypt(encCfg)
		if err != nil {
			return nil, fmt.Errorf("decrypt destination config id[%d]: %w", d.ID, err)
		}
		dests = append(dests, d)
	}
	if dests == nil {
		dests = []Destination{}
	}
	return dests, rows.Err()
}

// GetDestination returns a single destination by ID.
func (m *Manager) GetDestination(id int) (*Destination, error) {
	q := fmt.Sprintf(`
		SELECT d.id, d.name, d.dest_type, d.version, d.config, d.created_at, d.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by
		FROM %s d
		LEFT JOIN %s cu ON d.created_by_id = cu.id
		LEFT JOIN %s uu ON d.updated_by_id = uu.id
		WHERE d.id = $1 AND d.deleted_at IS NULL`,
		m.tbl("destination"), m.tbl("user"), m.tbl("user"))

	var d Destination
	var encCfg string
	err := m.db.QueryRowContext(context.Background(), q, id).Scan(
		&d.ID, &d.Name, &d.Type, &d.Version, &encCfg,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("destination not found id[%d]", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get destination: %w", err)
	}
	d.Config, err = m.decrypt(encCfg)
	if err != nil {
		return nil, fmt.Errorf("decrypt destination config: %w", err)
	}
	return &d, nil
}

// CreateDestination creates a new destination.
func (m *Manager) CreateDestination(d EntityBase) (*EntityBase, error) {
	unique, err := m.IsNameUnique("destination", d.Name)
	if err != nil {
		return nil, fmt.Errorf("check destination name uniqueness: %w", err)
	}
	if !unique {
		return nil, fmt.Errorf("destination name %q already exists in this project", d.Name)
	}
	encCfg, err := m.encrypt(d.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt destination config: %w", err)
	}
	q := fmt.Sprintf(`
		INSERT INTO %s (name, dest_type, version, config, project_id, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6, NOW(), NOW())`,
		m.tbl("destination"))
	_, err = m.db.ExecContext(context.Background(), q, d.Name, d.Type, d.Version, encCfg, m.projectID, m.currentUserID())
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}
	return &d, nil
}

// UpdateDestination updates an existing destination.
func (m *Manager) UpdateDestination(id int, d EntityBase) (*EntityBase, error) {
	encCfg, err := m.encrypt(d.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt destination config: %w", err)
	}
	q := fmt.Sprintf(`
		UPDATE %s SET name=$1, dest_type=$2, version=$3, config=$4, updated_by_id=$5, updated_at=NOW()
		WHERE id=$6`,
		m.tbl("destination"))
	_, err = m.db.ExecContext(context.Background(), q, d.Name, d.Type, d.Version, encCfg, m.currentUserID(), id)
	if err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}
	return &d, nil
}

// DeleteDestination soft-deletes a destination by ID (sets deleted_at = NOW()).
// Matches BFF Beego ORM soft-delete behavior.
func (m *Manager) DeleteDestination(id int) error {
	count, err := m.countJobsByDest(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("destination is used by %d job(s); delete those jobs first", count)
	}
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=NOW(), updated_at=NOW(), updated_by_id=$2 WHERE id=$1 AND deleted_at IS NULL`, m.tbl("destination"))
	m.authMu.RLock()
	uid := m.currentUserID()
	m.authMu.RUnlock()
	_, err = m.db.ExecContext(context.Background(), q, id, uid)
	return err
}

func (m *Manager) countJobsByDest(destID int) (int, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE dest_id=$1 AND deleted_at IS NULL`, m.tbl("job"))
	var n int
	err := m.db.QueryRowContext(context.Background(), q, destID).Scan(&n)
	return n, err
}

// ─── Jobs ─────────────────────────────────────────────────────────────────────

// ListJobs returns all jobs for the project.
func (m *Manager) ListJobs() ([]Job, error) {
	q := fmt.Sprintf(`
		SELECT j.id, j.name, j.frequency, j.active, j.advanced_settings,
		       j.created_at, j.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by,
		       s.id, s.name, s.type, s.version,
		       d.id, d.name, d.dest_type, d.version
		FROM %s j
		LEFT JOIN %s s ON j.source_id = s.id
		LEFT JOIN %s d ON j.dest_id = d.id
		LEFT JOIN %s cu ON j.created_by_id = cu.id
		LEFT JOIN %s uu ON j.updated_by_id = uu.id
		WHERE j.project_id = $1 AND j.deleted_at IS NULL
		ORDER BY j.updated_at DESC`,
		m.tbl("job"), m.tbl("source"), m.tbl("destination"), m.tbl("user"), m.tbl("user"))

	rows, err := m.db.QueryContext(context.Background(), q, m.projectID)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		j, err := m.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []Job{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Enrich jobs with real-time Temporal status (best-effort, 5s total cap).
	if m.temporal != nil && len(jobs) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for i := range jobs {
			select {
			case <-ctx.Done():
				break // timeout — use DB-cached values for remaining jobs
			default:
			}
			status, startTime, runType := m.fetchJobLastRun(jobs[i].ID)
			if status != "" {
				jobs[i].LastRunState = status
			}
			if startTime != "" {
				jobs[i].LastRunTime = startTime
			}
			if runType != "" {
				jobs[i].LastRunType = runType
			}
		}
	}

	return jobs, nil
}

// GetJob returns a single job by ID.
func (m *Manager) GetJob(id int) (*Job, error) {
	q := fmt.Sprintf(`
		SELECT j.id, j.name, j.frequency, j.active, j.advanced_settings,
		       j.created_at, j.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by,
		       s.id, s.name, s.type, s.version,
		       d.id, d.name, d.dest_type, d.version
		FROM %s j
		LEFT JOIN %s s ON j.source_id = s.id
		LEFT JOIN %s d ON j.dest_id = d.id
		LEFT JOIN %s cu ON j.created_by_id = cu.id
		LEFT JOIN %s uu ON j.updated_by_id = uu.id
		WHERE j.id = $1`,
		m.tbl("job"), m.tbl("source"), m.tbl("destination"), m.tbl("user"), m.tbl("user"))

	rows, err := m.db.QueryContext(context.Background(), q, id)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("job not found id[%d]", id)
	}
	j, err := m.scanJob(rows)
	if err != nil {
		return nil, err
	}
	return &j, rows.Err()
}

// scanJob scans a row from the jobs query into a Job struct.
func (m *Manager) scanJob(rows *sql.Rows) (Job, error) {
	var j Job
	var advRaw sql.NullString
	var srcID sql.NullInt64
	var srcName, srcType, srcVersion sql.NullString
	var dstID sql.NullInt64
	var dstName, dstType, dstVersion sql.NullString

	err := rows.Scan(
		&j.ID, &j.Name, &j.Frequency, &j.Activate, &advRaw,
		&j.CreatedAt, &j.UpdatedAt, &j.CreatedBy, &j.UpdatedBy,
		&srcID, &srcName, &srcType, &srcVersion,
		&dstID, &dstName, &dstType, &dstVersion,
	)
	if err != nil {
		return Job{}, fmt.Errorf("scan job: %w", err)
	}

	if srcID.Valid {
		j.Source = JobConnector{
			ID:      int(srcID.Int64),
			Name:    srcName.String,
			Type:    srcType.String,
			Version: srcVersion.String,
		}
	}
	if dstID.Valid {
		j.Destination = JobConnector{
			ID:      int(dstID.Int64),
			Name:    dstName.String,
			Type:    dstType.String,
			Version: dstVersion.String,
		}
	}
	if advRaw.Valid && advRaw.String != "" && advRaw.String != "{}" {
		var adv AdvancedSettings
		if err := json.Unmarshal([]byte(advRaw.String), &adv); err == nil {
			j.AdvancedSettings = &adv
		}
	}
	return j, nil
}

// DeleteJob deletes a job by ID, and removes its Temporal schedule if connected.
func (m *Manager) DeleteJob(id int) error {
	job, err := m.GetJob(id)
	if err != nil {
		return err
	}

	// Best-effort: remove Temporal schedule
	if m.temporal != nil {
		_, scheduleID := m.workflowAndScheduleID(m.projectID, id)
		handle := m.temporal.ScheduleClient().GetHandle(context.Background(), scheduleID)
		_ = handle.Delete(context.Background()) // ignore error if not found
	}

	// Soft-delete the job record to match BFF behavior (sets deleted_at = NOW()).
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=NOW(), updated_at=NOW(), updated_by_id=$2 WHERE id=$1 AND deleted_at IS NULL`, m.tbl("job"))
	m.authMu.RLock()
	uid := m.currentUserID()
	m.authMu.RUnlock()
	_, err = m.db.ExecContext(context.Background(), q, id, uid)
	_ = job // referenced above
	return err
}

// TriggerSync triggers an immediate sync for a job via Temporal schedule.
func (m *Manager) TriggerSync(id int) error {
	if m.temporal == nil {
		return fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}
	_, scheduleID := m.workflowAndScheduleID(m.projectID, id)
	handle := m.temporal.ScheduleClient().GetHandle(context.Background(), scheduleID)
	return handle.Trigger(context.Background(), client.ScheduleTriggerOptions{})
}

// CancelJob cancels a running workflow for a job.
func (m *Manager) CancelJob(id int) error {
	if m.temporal == nil {
		return fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}
	workflowID, _ := m.workflowAndScheduleID(m.projectID, id)
	// Cancel any running workflow with this ID prefix
	return m.temporal.CancelWorkflow(context.Background(), workflowID, "")
}

// ActivateJob pauses or resumes a job's Temporal schedule and updates the DB.
func (m *Manager) ActivateJob(id int, activate bool) error {
	if m.temporal != nil {
		_, scheduleID := m.workflowAndScheduleID(m.projectID, id)
		handle := m.temporal.ScheduleClient().GetHandle(context.Background(), scheduleID)
		if activate {
			_ = handle.Unpause(context.Background(), client.ScheduleUnpauseOptions{Note: "user resumed"})
		} else {
			_ = handle.Pause(context.Background(), client.SchedulePauseOptions{Note: "user paused"})
		}
	}

	q := fmt.Sprintf(`UPDATE %s SET active=$1, updated_by_id=$2, updated_at=NOW() WHERE id=$3 AND deleted_at IS NULL`, m.tbl("job"))
	_, err := m.db.ExecContext(context.Background(), q, activate, m.currentUserID(), id)
	return err
}

// ListJobTasks returns the run history for a job from Temporal.
func (m *Manager) ListJobTasks(jobID int) ([]JobTask, error) {
	if m.temporal == nil {
		return nil, fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}

	query := fmt.Sprintf("WorkflowId BETWEEN 'sync-%s-%d-' AND 'sync-%s-%d-z'",
		m.projectID, jobID, m.projectID, jobID)

	resp, err := m.temporal.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	var tasks []JobTask
	for _, execution := range resp.GetExecutions() {
		startTime := execution.GetStartTime().AsTime().UTC()
		var runTime string
		if ct := execution.GetCloseTime(); ct != nil {
			runTime = ct.AsTime().UTC().Sub(startTime).Round(time.Second).String()
		} else {
			runTime = time.Since(startTime).Round(time.Second).String()
		}
		tasks = append(tasks, JobTask{
			Runtime:   runTime,
			StartTime: startTime.Format(time.RFC3339),
			Status:    execution.GetStatus().String(),
			FilePath:  execution.GetExecution().GetWorkflowId(),
			JobType:   "sync",
		})
	}
	return tasks, nil
}

// GetTaskLogs returns paginated log entries for a job task by reading the
// worker log files written to the shared config volume.
func (m *Manager) GetTaskLogs(jobID int, taskID string, filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error) {
	if _, err := m.GetJob(jobID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("log file path is required")
	}
	return readTaskLogsFromDisk(filePath, cursor, limit, direction)
}

// GetSettings returns project-level settings.
func (m *Manager) GetSettings() (*SystemSettings, error) {
	q := fmt.Sprintf(`
		SELECT id, project_id, COALESCE(webhook_alert_url,'') FROM %s WHERE project_id=$1 LIMIT 1`,
		m.tbl("project-settings"))
	var s SystemSettings
	err := m.db.QueryRowContext(context.Background(), q, m.projectID).Scan(&s.ID, &s.ProjectID, &s.WebhookAlertURL)
	if err == sql.ErrNoRows {
		return &SystemSettings{ProjectID: m.projectID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

// UpdateSettings saves project-level settings.
func (m *Manager) UpdateSettings(s SystemSettings) error {
	q := fmt.Sprintf(`
		INSERT INTO %s (project_id, webhook_alert_url, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (project_id) DO UPDATE SET webhook_alert_url=EXCLUDED.webhook_alert_url, updated_at=NOW()`,
		m.tbl("project-settings"))
	_, err := m.db.ExecContext(context.Background(), q, m.projectID, s.WebhookAlertURL)
	return err
}

// UpdateJobMeta updates a job's name and frequency (schedule).
func (m *Manager) UpdateJobMeta(id int, name, frequency string) error {
	q := fmt.Sprintf(`UPDATE %s SET name=$1, frequency=$2, updated_by_id=$3, updated_at=NOW() WHERE id=$4 AND deleted_at IS NULL`, m.tbl("job"))
	_, err := m.db.ExecContext(context.Background(), q, name, frequency, m.currentUserID(), id)
	return err
}

// ─── Temporal filesystem helpers ─────────────────────────────────────────────
//
// These mirror the BFF's internal/services/temporal/filesystem.go.
// Direct (one-shot) commands (discover, check) use the raw workflowID as the
// subdirectory; async (scheduled) commands (sync, clear-destination) use a
// SHA-256 hash of the workflowID.

// temporalWorkDir returns the absolute work directory for a given command +
// workflowID, matching the BFF's getWorkflowDirectory logic.
func temporalWorkDir(cmd temporalCommand, workflowID string) string {
	var subDir string
	switch cmd {
	case cmdSync, cmdClearDestination:
		h := sha256.Sum256([]byte(workflowID))
		subDir = fmt.Sprintf("%x", h)
	default:
		subDir = workflowID
	}
	return filepath.Join(DefaultConfigDir, subDir)
}

// setupConfigFiles creates the work directory and writes named config files into it.
func setupConfigFiles(cmd temporalCommand, workflowID string, configs []jobConfig) error {
	workDir := temporalWorkDir(cmd, workflowID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("create config dir %s: %w", workDir, err)
	}
	for _, cfg := range configs {
		p := filepath.Join(workDir, cfg.Name)
		if err := os.WriteFile(p, []byte(cfg.Data), 0644); err != nil {
			return fmt.Errorf("write %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// readJSONFile reads a JSON file and unmarshals it into a map.
func readJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return result, nil
}

// extractWorkflowResponse calls run.Get() to retrieve the relative file path
// returned by the worker, then reads and returns the JSON payload from disk.
func extractWorkflowResponse(ctx context.Context, run client.WorkflowRun) (map[string]interface{}, error) {
	raw := make(map[string]interface{})
	if err := run.Get(ctx, &raw); err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}
	response, ok := raw["response"].(string)
	if !ok || response == "" {
		return nil, fmt.Errorf("invalid response format from worker (missing 'response' key)")
	}
	responsePath := filepath.Join(DefaultConfigDir, response)
	return readJSONFile(responsePath)
}

// cleanupWorkDir removes the temporary config directory for a workflow.
func cleanupWorkDir(cmd temporalCommand, workflowID string) {
	_ = os.RemoveAll(temporalWorkDir(cmd, workflowID))
}

// ─── Temporal one-shot workflow executor ─────────────────────────────────────

// executeWorkflow starts an ExecuteWorkflow Temporal workflow and waits for
// the result, returning the JSON payload from the output file.
func (m *Manager) executeWorkflow(ctx context.Context, req executionRequest) (map[string]interface{}, error) {
	if m.temporal == nil {
		return nil, fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}
	opts := client.StartWorkflowOptions{
		ID:        req.WorkflowID,
		TaskQueue: TemporalTaskQueue,
	}
	run, err := m.temporal.ExecuteWorkflow(ctx, opts, WorkflowTypeExecute, req)
	if err != nil {
		return nil, fmt.Errorf("start workflow %s: %w", req.WorkflowID, err)
	}
	return extractWorkflowResponse(ctx, run)
}

// ─── ClearDestination ────────────────────────────────────────────────────────

// ClearDestination triggers a clear-destination workflow for a job.
//
// Implementation mirrors the BFF's ETLService.ClearDestination:
//  1. Fetches the job and its streams_config from the DB.
//  2. Pauses the Temporal schedule to prevent a race with a new sync.
//  3. Waits up to cancelSyncWaitTimeout for any running sync to finish.
//  4. Writes the streams catalog to a temp file on the shared volume.
//  5. Updates the Temporal schedule to run RunSyncWorkflow with a
//     clear-destination ExecutionRequest.
//  6. Triggers the schedule immediately (one-shot).
//  7. On trigger failure, restores the schedule back to the normal sync request.
func (m *Manager) ClearDestination(jobID int) error {
	if m.temporal == nil {
		return fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}

	job, err := m.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if !job.Activate {
		return fmt.Errorf("job is paused — unpause the job before running clear-destination")
	}

	workflowID, scheduleID := m.workflowAndScheduleID(m.projectID, jobID)
	ctx := context.Background()

	// 1. Pause the schedule to prevent a new sync starting while we prepare.
	handle := m.temporal.ScheduleClient().GetHandle(ctx, scheduleID)
	if err := handle.Pause(ctx, client.SchedulePauseOptions{Note: "clear-destination in progress"}); err != nil {
		return fmt.Errorf("pause schedule: %w", err)
	}

	// 2. Wait for any running sync to finish (best-effort, limited window).
	waitCtx, cancel := context.WithTimeout(ctx, cancelSyncWaitTimeout)
	defer cancel()
	m.waitForSyncToStop(waitCtx, workflowID)

	// 3. Build the clear-destination execution request.
	clearReq, tempRelPath, err := m.buildClearDestRequest(job, workflowID)
	if err != nil {
		// Resume schedule before returning error.
		_ = handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "clear-destination setup failed"})
		return fmt.Errorf("build clear-destination request: %w", err)
	}

	// 4. Update the schedule to use the clear-destination request.
	if err := m.updateSchedule(ctx, job.Frequency, jobID, clearReq); err != nil {
		_ = handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "clear-destination update failed"})
		if tempRelPath != "" {
			_ = os.Remove(filepath.Join(DefaultConfigDir, tempRelPath))
		}
		return fmt.Errorf("update schedule for clear-destination: %w", err)
	}

	// 5. Trigger the schedule immediately.
	if err := handle.Trigger(ctx, client.ScheduleTriggerOptions{
		Overlap: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
	}); err != nil {
		// Revert schedule back to normal sync.
		syncReq := m.buildSyncRequest(job, workflowID)
		_ = m.updateSchedule(ctx, job.Frequency, jobID, syncReq)
		_ = handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "clear-destination trigger failed, reverted"})
		if tempRelPath != "" {
			_ = os.Remove(filepath.Join(DefaultConfigDir, tempRelPath))
		}
		return fmt.Errorf("trigger clear-destination schedule: %w", err)
	}

	return nil
}

// buildSyncRequest builds the normal RunSyncWorkflow ExecutionRequest (matches BFF).
func (m *Manager) buildSyncRequest(job *Job, workflowID string) *executionRequest {
	return &executionRequest{
		Command:       cmdSync,
		ConnectorType: job.Source.Type,
		Version:       job.Source.Version,
		Args: []string{
			"sync",
			"--config", "/mnt/config/source.json",
			"--destination", "/mnt/config/destination.json",
			"--catalog", "/mnt/config/streams.json",
			"--state", "/mnt/config/state.json",
		},
		WorkflowID: workflowID,
		JobID:      job.ID,
		ProjectID:  m.projectID,
		Timeout:    syncTimeout,
		OutputFile: "state.json",
	}
}

// buildClearDestRequest builds the RunSyncWorkflow ExecutionRequest for a
// clear-destination run, writing the streams catalog to the shared volume.
// Returns the request and the relative path of the written temp file (for cleanup).
func (m *Manager) buildClearDestRequest(job *Job, workflowID string) (*executionRequest, string, error) {
	catalog := job.StreamsConfig

	// Write the streams catalog to a unique subdirectory so the worker can read it.
	streamsDir := fmt.Sprintf("%s-%d", workflowID, time.Now().Unix())
	relativePath := filepath.Join(streamsDir, "streams.json")
	streamsPath := filepath.Join(DefaultConfigDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(streamsPath), 0755); err != nil {
		return nil, "", fmt.Errorf("create streams dir: %w", err)
	}
	if err := os.WriteFile(streamsPath, []byte(catalog), 0644); err != nil {
		return nil, "", fmt.Errorf("write streams file: %w", err)
	}

	req := &executionRequest{
		Command:       cmdClearDestination,
		ConnectorType: job.Source.Type,
		Version:       job.Source.Version,
		Args: []string{
			"clear-destination",
			"--streams", "/mnt/config/streams.json",
			"--state", "/mnt/config/state.json",
			"--destination", "/mnt/config/destination.json",
		},
		WorkflowID: workflowID,
		ProjectID:  m.projectID,
		JobID:      job.ID,
		Timeout:    clearDestTimeout,
		OutputFile: "state.json",
		TempPath:   relativePath,
	}
	return req, relativePath, nil
}

// updateSchedule updates a Temporal schedule's workflow action.
func (m *Manager) updateSchedule(ctx context.Context, frequency string, jobID int, req *executionRequest) error {
	_, scheduleID := m.workflowAndScheduleID(m.projectID, jobID)
	handle := m.temporal.ScheduleClient().GetHandle(ctx, scheduleID)
	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			if frequency != "" {
				input.Description.Schedule.Spec = &client.ScheduleSpec{
					CronExpressions: []string{frequency},
				}
			}
			if req != nil {
				input.Description.Schedule.Action = &client.ScheduleWorkflowAction{
					ID:        req.WorkflowID,
					Workflow:  WorkflowTypeRunSync,
					Args:      []any{*req},
					TaskQueue: TemporalTaskQueue,
				}
			}
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
}

// waitForSyncToStop polls Temporal for running sync workflows and waits until
// they finish or the context is cancelled.
func (m *Manager) waitForSyncToStop(ctx context.Context, workflowID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q := fmt.Sprintf("WorkflowId = '%s' AND ExecutionStatus = 'Running'", workflowID)
			resp, err := m.temporal.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
				Query: q,
			})
			if err != nil || len(resp.GetExecutions()) == 0 {
				return
			}
		}
	}
}

// ─── TestSource ──────────────────────────────────────────────────────────────

// TestSource tests a source connection by triggering a Temporal check workflow.
//
// Mirrors BFF's ETLService.TestSourceConnection:
//  1. Encrypts the connector config (so the worker can decrypt it).
//  2. Writes config.json to the shared volume at /tmp/olake-config/<workflowID>/.
//  3. Executes the ExecuteWorkflow Temporal workflow with `check --config`.
//  4. Reads the connection result from the output file.
func (m *Manager) TestSource(s EntityBase) (*TestConnectionResult, error) {
	if m.temporal == nil {
		return nil, fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}

	workflowID := fmt.Sprintf("test-connection-%s-%d", s.Type, time.Now().Unix())

	// Encrypt config before sending to the worker (matches BFF behaviour).
	encryptedConfig, err := m.encrypt(s.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt source config: %w", err)
	}

	configs := []jobConfig{
		{Name: "config.json", Data: encryptedConfig},
	}
	if err := setupConfigFiles(cmdCheck, workflowID, configs); err != nil {
		return nil, fmt.Errorf("setup config files: %w", err)
	}
	defer cleanupWorkDir(cmdCheck, workflowID)

	cmdArgs := []string{
		"check",
		"--config", "/mnt/config/config.json",
	}
	// Pass encryption key if configured so the worker can decrypt the config.
	if m.encryptionKey != "" {
		cmdArgs = append(cmdArgs, "--encryption-key", m.encryptionKey)
	}

	req := executionRequest{
		Command:       cmdCheck,
		ConnectorType: s.Type,
		Version:       s.Version,
		Args:          cmdArgs,
		WorkflowID:    workflowID,
		Timeout:       checkTimeout,
		OutputFile:    "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout+30*time.Second)
	defer cancel()

	result, err := m.executeWorkflow(ctx, req)
	if err != nil {
		return &TestConnectionResult{
			ConnectionResult: struct {
				Message string `json:"message"`
				Status  string `json:"status"`
			}{Message: err.Error(), Status: "FAILED"},
		}, fmt.Errorf("check workflow: %w", err)
	}

	return parseConnectionResult(result), nil
}

// TestDestination tests a destination connection via Temporal.
//
// Mirrors BFF's ETLService.TestDestinationConnection.
// The flag passed to `olake check` is "--destination" (not "--config").
func (m *Manager) TestDestination(d EntityBase, sourceType, sourceVersion string) (*TestConnectionResult, error) {
	if m.temporal == nil {
		return nil, fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}

	workflowID := fmt.Sprintf("test-connection-%s-%d", d.Type, time.Now().Unix())

	encryptedConfig, err := m.encrypt(d.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt destination config: %w", err)
	}

	configs := []jobConfig{
		{Name: "config.json", Data: encryptedConfig},
	}
	if err := setupConfigFiles(cmdCheck, workflowID, configs); err != nil {
		return nil, fmt.Errorf("setup config files: %w", err)
	}
	defer cleanupWorkDir(cmdCheck, workflowID)

	cmdArgs := []string{
		"check",
		"--destination", "/mnt/config/config.json",
	}
	if m.encryptionKey != "" {
		cmdArgs = append(cmdArgs, "--encryption-key", m.encryptionKey)
	}

	// Use source driver to run the check command (matches BFF).
	connectorType := sourceType
	if connectorType == "" {
		connectorType = "postgres" // sensible fallback
	}
	version := sourceVersion
	if version == "" {
		version = d.Version
	}

	req := executionRequest{
		Command:       cmdCheck,
		ConnectorType: connectorType,
		Version:       version,
		Args:          cmdArgs,
		WorkflowID:    workflowID,
		Timeout:       checkTimeout,
		OutputFile:    "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout+30*time.Second)
	defer cancel()

	result, err := m.executeWorkflow(ctx, req)
	if err != nil {
		return &TestConnectionResult{
			ConnectionResult: struct {
				Message string `json:"message"`
				Status  string `json:"status"`
			}{Message: err.Error(), Status: "FAILED"},
		}, fmt.Errorf("check workflow: %w", err)
	}

	return parseConnectionResult(result), nil
}

// parseConnectionResult extracts a TestConnectionResult from a workflow response map.
func parseConnectionResult(result map[string]interface{}) *TestConnectionResult {
	out := &TestConnectionResult{}
	cs, ok := result["connectionStatus"].(map[string]interface{})
	if !ok {
		cs = result // some workers return flat map
	}
	if status, ok := cs["status"].(string); ok {
		out.ConnectionResult.Status = status
	}
	if msg, ok := cs["message"].(string); ok {
		out.ConnectionResult.Message = msg
	}
	return out
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// workflowAndScheduleID matches the BFF's naming convention.
func (m *Manager) workflowAndScheduleID(projectID string, jobID int) (string, string) {
	workflowID := fmt.Sprintf("sync-%s-%d", projectID, jobID)
	return workflowID, fmt.Sprintf("schedule-%s", workflowID)
}

// ─── Streams / Discover ──────────────────────────────────────────────────────

// StreamInfo describes a single stream returned by discover.
type StreamInfo struct {
	Namespace    string   `json:"namespace"`
	Name         string   `json:"name"`
	SyncModes    []string `json:"supported_sync_modes"`
	CursorFields []string `json:"available_cursor_fields"`
}

// StreamConfig holds the per-stream configuration selected by the user.
type StreamConfig struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	SyncMode    string `json:"sync_mode"`
	CursorField string `json:"cursor_field,omitempty"`
	Normalize   bool   `json:"normalize"`
	Selected    bool   `json:"selected"`
}

// DiscoverStreams returns the stream catalogue for a source by triggering a
// Temporal discover workflow, matching the BFF's GetSourceCatalog behaviour.
//
// Flow (mirrors BFF's temporal.DiscoverStreams):
//  1. Fetches and decrypts the source config from the database.
//  2. Re-encrypts it for the worker (the worker expects encrypted config).
//  3. Writes config.json (and an empty streams.json) to /tmp/olake-config/<workflowID>/.
//  4. Executes the ExecuteWorkflow Temporal workflow with `discover --config …`.
//  5. Reads the resulting streams.json output file returned by the worker.
//  6. Parses the catalog into []StreamInfo.
func (m *Manager) DiscoverStreams(sourceID int) ([]StreamInfo, error) {
	if m.temporal == nil {
		return nil, fmt.Errorf("temporal client not connected — set TEMPORAL_ADDRESS")
	}

	src, err := m.GetSource(sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source id[%d]: %w", sourceID, err)
	}

	// src.Config is already decrypted by GetSource; re-encrypt for the worker.
	encryptedConfig, err := m.encrypt(src.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt source config: %w", err)
	}

	workflowID := fmt.Sprintf("discover-catalog-%s-%d", src.Type, time.Now().Unix())

	configs := []jobConfig{
		{Name: "config.json", Data: encryptedConfig},
		{Name: "streams.json", Data: ""},
	}
	if err := setupConfigFiles(cmdDiscover, workflowID, configs); err != nil {
		return nil, fmt.Errorf("setup config files: %w", err)
	}
	defer cleanupWorkDir(cmdDiscover, workflowID)

	cmdArgs := []string{
		"discover",
		"--config", "/mnt/config/config.json",
	}
	if m.encryptionKey != "" {
		cmdArgs = append(cmdArgs, "--encryption-key", m.encryptionKey)
	}

	req := executionRequest{
		Command:       cmdDiscover,
		ConnectorType: src.Type,
		Version:       src.Version,
		Args:          cmdArgs,
		WorkflowID:    workflowID,
		Timeout:       discoverTimeout,
		OutputFile:    "streams.json",
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoverTimeout+30*time.Second)
	defer cancel()

	result, err := m.executeWorkflow(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("discover workflow: %w", err)
	}

	return parseStreamsFromCatalog(result), nil
}

// parseStreamsFromCatalog extracts []StreamInfo from the JSON map returned by
// the olake discover command.  The BFF returns the raw map to the frontend;
// the TUI needs to parse it into typed structs.
//
// The olake catalog JSON has this structure:
//
//	{
//	  "streams": [
//	    {
//	      "stream": {
//	        "name": "orders",
//	        "namespace": "public",
//	        "supported_sync_modes": ["full_refresh", "incremental"],
//	        "source_defined_cursor": true,
//	        "default_cursor_field": ["updated_at"]
//	      },
//	      "sync_mode": "incremental",
//	      "cursor_field": ["updated_at"]
//	    },
//	    ...
//	  ]
//	}
func parseStreamsFromCatalog(result map[string]interface{}) []StreamInfo {
	streamsRaw, ok := result["streams"].([]interface{})
	if !ok {
		return nil
	}
	var out []StreamInfo
	for _, item := range streamsRaw {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		streamObj, ok := entry["stream"].(map[string]interface{})
		if !ok {
			// Some workers return the fields directly at the top level.
			streamObj = entry
		}
		var si StreamInfo
		si.Name, _ = streamObj["name"].(string)
		si.Namespace, _ = streamObj["namespace"].(string)

		if modes, ok := streamObj["supported_sync_modes"].([]interface{}); ok {
			for _, m := range modes {
				if s, ok := m.(string); ok {
					si.SyncModes = append(si.SyncModes, s)
				}
			}
		}
		if cursors, ok := streamObj["default_cursor_field"].([]interface{}); ok {
			for _, c := range cursors {
				if s, ok := c.(string); ok {
					si.CursorFields = append(si.CursorFields, s)
				}
			}
		}
		if len(si.CursorFields) == 0 {
			// Fallback: check available_cursor_fields
			if cursors, ok := streamObj["available_cursor_fields"].([]interface{}); ok {
				for _, c := range cursors {
					if s, ok := c.(string); ok {
						si.CursorFields = append(si.CursorFields, s)
					}
				}
			}
		}
		out = append(out, si)
	}
	return out
}

// CreateJob inserts a new job record into the database.
func (m *Manager) CreateJob(name string, sourceID, destID int, frequency string, streams []StreamConfig) (*Job, error) {
	if frequency == "" {
		frequency = "0 * * * *" // every hour default
	}

	unique, err := m.IsNameUnique("job", name)
	if err != nil {
		return nil, fmt.Errorf("check job name uniqueness: %w", err)
	}
	if !unique {
		return nil, fmt.Errorf("job name %q already exists in this project", name)
	}

	streamsJSON, err := json.Marshal(streams)
	if err != nil {
		return nil, fmt.Errorf("marshal streams: %w", err)
	}

	q := fmt.Sprintf(`
		INSERT INTO %s
		  (name, source_id, dest_id, frequency, active, streams_config,
		   project_id, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8,NOW(),NOW())
		RETURNING id`,
		m.tbl("job"))

	var jobID int
	err = m.db.QueryRowContext(context.Background(), q,
		name, sourceID, destID, frequency, true, string(streamsJSON),
		m.projectID, m.currentUserID(),
	).Scan(&jobID)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	// Create Temporal schedule so the job actually runs on its cron frequency.
	// Best-effort: if Temporal is not connected the job exists in DB but won't
	// auto-sync until manually triggered or Temporal reconnects.
	if m.temporal != nil {
		job, err := m.GetJob(jobID)
		if err != nil {
			return nil, fmt.Errorf("get created job for schedule: %w", err)
		}

		src, srcErr := m.GetSource(job.Source.ID)
		dst, dstErr := m.GetDestination(job.Destination.ID)
		if srcErr == nil && dstErr == nil {
			workflowID, scheduleID := m.workflowAndScheduleID(m.projectID, jobID)

			encSrcCfg, _ := m.encrypt(src.Config)
			encDstCfg, _ := m.encrypt(dst.Config)

			configs := []jobConfig{
				{Name: "source.json", Data: encSrcCfg},
				{Name: "destination.json", Data: encDstCfg},
				{Name: "streams.json", Data: string(streamsJSON)},
				{Name: "state.json", Data: "{}"},
			}
			_ = setupConfigFiles(cmdSync, workflowID, configs)

			syncReq := executionRequest{
				Command:       cmdSync,
				ConnectorType: job.Source.Type,
				Version:       job.Source.Version,
				Args: []string{
					"sync",
					"--config", "/mnt/config/source.json",
					"--destination", "/mnt/config/destination.json",
					"--catalog", "/mnt/config/streams.json",
					"--state", "/mnt/config/state.json",
				},
				WorkflowID: workflowID,
				JobID:      jobID,
				ProjectID:  m.projectID,
				Timeout:    syncTimeout,
				OutputFile: "state.json",
			}
			if m.encryptionKey != "" {
				syncReq.Args = append(syncReq.Args, "--encryption-key", m.encryptionKey)
			}

			_, _ = m.temporal.ScheduleClient().Create(context.Background(), client.ScheduleOptions{
				ID: scheduleID,
				Spec: client.ScheduleSpec{
					CronExpressions: []string{frequency},
				},
				Action: &client.ScheduleWorkflowAction{
					ID:        workflowID,
					Workflow:  WorkflowTypeRunSync,
					Args:      []any{syncReq},
					TaskQueue: TemporalTaskQueue,
				},
			})
		}

		return job, nil
	}

	return m.GetJob(jobID)
}

// ─── Name Uniqueness ─────────────────────────────────────────────────────────

// IsNameUnique checks whether a name is unique within the project for a given
// entity type ("job", "source", or "destination").
func (m *Manager) IsNameUnique(entityType string, name string) (bool, error) {
	var table string
	switch entityType {
	case "job":
		table = m.tbl("job")
	case "source":
		table = m.tbl("source")
	case "destination":
		table = m.tbl("destination")
	default:
		return false, fmt.Errorf("invalid entity type: %s", entityType)
	}
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE name=$1 AND project_id=$2 AND deleted_at IS NULL`, table)
	var count int
	err := m.db.QueryRowContext(context.Background(), q, name, m.projectID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check name uniqueness: %w", err)
	}
	return count == 0, nil
}

// ─── Clear Destination Status & Recovery ─────────────────────────────────────

// GetClearDestStatus reports whether a clear-destination workflow is currently
// running for the given job.
func (m *Manager) GetClearDestStatus(jobID int) (bool, error) {
	if m.temporal == nil {
		return false, fmt.Errorf("temporal client not connected")
	}
	workflowID, _ := m.workflowAndScheduleID(m.projectID, jobID)
	query := fmt.Sprintf("WorkflowId = '%s' AND ExecutionStatus = 'Running'", workflowID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := m.temporal.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{Query: query})
	if err != nil {
		return false, fmt.Errorf("list workflows: %w", err)
	}
	for _, exec := range resp.GetExecutions() {
		if exec.GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return true, nil
		}
	}
	return false, nil
}

// RecoverFromClearDest cancels stuck clear-destination workflows and restores
// the normal sync schedule for a job.
func (m *Manager) RecoverFromClearDest(jobID int) error {
	if m.temporal == nil {
		return fmt.Errorf("temporal client not connected")
	}

	job, err := m.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}

	workflowID, scheduleID := m.workflowAndScheduleID(m.projectID, jobID)
	ctx := context.Background()

	// 1. Cancel any running workflows for this job.
	_ = m.temporal.CancelWorkflow(ctx, workflowID, "")

	// 2. Restore schedule to normal sync workflow.
	syncReq := m.buildSyncRequest(job, workflowID)
	if err := m.updateSchedule(ctx, job.Frequency, jobID, syncReq); err != nil {
		return fmt.Errorf("restore sync schedule: %w", err)
	}

	// 3. Resume the schedule.
	handle := m.temporal.ScheduleClient().GetHandle(ctx, scheduleID)
	if err := handle.Unpause(ctx, client.ScheduleUnpauseOptions{Note: "recovered from clear-destination"}); err != nil {
		return fmt.Errorf("resume schedule: %w", err)
	}

	return nil
}

// ─── Full Job Update ─────────────────────────────────────────────────────────

// UpdateJobFull updates all job fields, matching the BFF's UpdateJob behaviour.
// It blocks when clear-destination is running, cancels any in-flight sync, and
// updates the Temporal schedule if the frequency changes.
func (m *Manager) UpdateJobFull(id int, name string, sourceID, destID int, frequency string, streams []StreamConfig, activate bool, advancedSettings *AdvancedSettings) error {
	// 1. Check if clear-destination is running.
	if m.temporal != nil {
		running, err := m.GetClearDestStatus(id)
		if err == nil && running {
			return fmt.Errorf("clear-destination is in progress, cannot update job")
		}
	}

	// 2. Cancel any running sync workflows.
	if m.temporal != nil {
		workflowID, _ := m.workflowAndScheduleID(m.projectID, id)
		_ = m.temporal.CancelWorkflow(context.Background(), workflowID, "")
	}

	// 3. Serialize streams and advanced settings.
	streamsJSON, err := json.Marshal(streams)
	if err != nil {
		return fmt.Errorf("marshal streams: %w", err)
	}
	var advJSON sql.NullString
	if advancedSettings != nil {
		b, err := json.Marshal(advancedSettings)
		if err != nil {
			return fmt.Errorf("marshal advanced settings: %w", err)
		}
		advJSON = sql.NullString{String: string(b), Valid: true}
	}

	// 4. Update all fields in DB.
	q := fmt.Sprintf(`
		UPDATE %s
		SET name=$1, source_id=$2, dest_id=$3, frequency=$4,
		    streams_config=$5, active=$6, advanced_settings=$7,
		    updated_by_id=$8, updated_at=NOW()
		WHERE id=$9 AND deleted_at IS NULL`, m.tbl("job"))
	_, err = m.db.ExecContext(context.Background(), q,
		name, sourceID, destID, frequency,
		string(streamsJSON), activate, advJSON,
		m.currentUserID(), id)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}

	// 5. Update Temporal schedule if frequency changed.
	if m.temporal != nil {
		job, err := m.GetJob(id)
		if err == nil {
			workflowID, _ := m.workflowAndScheduleID(m.projectID, id)
			syncReq := m.buildSyncRequest(job, workflowID)
			_ = m.updateSchedule(context.Background(), frequency, id, syncReq)
		}
	}

	return nil
}

// ─── Real-time Job Status from Temporal ──────────────────────────────────────

// fetchJobLastRun queries Temporal for the latest workflow execution of a job
// and returns the status, start time, and run type. Returns zero values on any
// error so callers can silently fall back to DB-cached values.
func (m *Manager) fetchJobLastRun(jobID int) (status, startTime, runType string) {
	if m.temporal == nil {
		return "", "", ""
	}
	workflowID := fmt.Sprintf("sync-%s-%d", m.projectID, jobID)
	query := fmt.Sprintf("WorkflowId BETWEEN '%s-' AND '%s-z'", workflowID, workflowID)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := m.temporal.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{Query: query})
	if err != nil || len(resp.GetExecutions()) == 0 {
		return "", "", ""
	}
	exec := resp.GetExecutions()[0]
	st := exec.GetStartTime().AsTime().UTC().Format(time.RFC3339)
	return exec.GetStatus().String(), st, "sync"
}
