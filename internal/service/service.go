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
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultProjectID    = "123"
	DefaultTemporalHost = "localhost:7233"
	DefaultRunMode      = "dev"
)

// ─── Domain types (kept identical to the HTTP version) ─────────────────────

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

// ─── Sources ─────────────────────────────────────────────────────────────────

// ListSources returns all sources for the project.
func (m *Manager) ListSources() ([]Source, error) {
	q := fmt.Sprintf(`
		SELECT s.id, s.name, s.type, s.version, s.config, s.created_at, s.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by
		FROM %s s
		LEFT JOIN %s cu ON s.created_by_id = cu.id
		LEFT JOIN %s uu ON s.updated_by_id = uu.id
		WHERE s.project_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.updated_at DESC`,
		m.tbl("source"), m.tbl("user"), m.tbl("user"))

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
			&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy); err != nil {
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
	encCfg, err := m.encrypt(s.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt source config: %w", err)
	}
	q := fmt.Sprintf(`
		INSERT INTO %s (name, type, version, config, project_id, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6, NOW(), NOW())`,
		m.tbl("source"))
	_, err = m.db.ExecContext(context.Background(), q, s.Name, s.Type, s.Version, encCfg, m.projectID, m.userID)
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
	_, err = m.db.ExecContext(context.Background(), q, s.Name, s.Type, s.Version, encCfg, m.userID, id)
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
	m.authMu.RLock()
	uid := m.userID
	m.authMu.RUnlock()
	_, err = m.db.ExecContext(context.Background(), q, id, uid)
	return err
}

func (m *Manager) countJobsBySource(sourceID int) (int, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE source_id=$1`, m.tbl("job"))
	var n int
	err := m.db.QueryRowContext(context.Background(), q, sourceID).Scan(&n)
	return n, err
}

// ─── Destinations ─────────────────────────────────────────────────────────────

// ListDestinations returns all destinations for the project.
func (m *Manager) ListDestinations() ([]Destination, error) {
	q := fmt.Sprintf(`
		SELECT d.id, d.name, d.type, d.version, d.config, d.created_at, d.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by
		FROM %s d
		LEFT JOIN %s cu ON d.created_by_id = cu.id
		LEFT JOIN %s uu ON d.updated_by_id = uu.id
		WHERE d.project_id = $1 AND d.deleted_at IS NULL
		ORDER BY d.updated_at DESC`,
		m.tbl("destination"), m.tbl("user"), m.tbl("user"))

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
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy); err != nil {
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
		SELECT d.id, d.name, d.type, d.version, d.config, d.created_at, d.updated_at,
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
	encCfg, err := m.encrypt(d.Config)
	if err != nil {
		return nil, fmt.Errorf("encrypt destination config: %w", err)
	}
	q := fmt.Sprintf(`
		INSERT INTO %s (name, type, version, config, project_id, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6, NOW(), NOW())`,
		m.tbl("destination"))
	_, err = m.db.ExecContext(context.Background(), q, d.Name, d.Type, d.Version, encCfg, m.projectID, m.userID)
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
		UPDATE %s SET name=$1, type=$2, version=$3, config=$4, updated_by_id=$5, updated_at=NOW()
		WHERE id=$6`,
		m.tbl("destination"))
	_, err = m.db.ExecContext(context.Background(), q, d.Name, d.Type, d.Version, encCfg, m.userID, id)
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
	uid := m.userID
	m.authMu.RUnlock()
	_, err = m.db.ExecContext(context.Background(), q, id, uid)
	return err
}

func (m *Manager) countJobsByDest(destID int) (int, error) {
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE dest_id=$1`, m.tbl("job"))
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
		       d.id, d.name, d.type, d.version
		FROM %s j
		LEFT JOIN %s s ON j.source_id = s.id
		LEFT JOIN %s d ON j.dest_id = d.id
		LEFT JOIN %s cu ON j.created_by_id = cu.id
		LEFT JOIN %s uu ON j.updated_by_id = uu.id
		WHERE j.project_id = $1
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
	return jobs, rows.Err()
}

// GetJob returns a single job by ID.
func (m *Manager) GetJob(id int) (*Job, error) {
	q := fmt.Sprintf(`
		SELECT j.id, j.name, j.frequency, j.active, j.advanced_settings,
		       j.created_at, j.updated_at,
		       COALESCE(cu.username,'') AS created_by, COALESCE(uu.username,'') AS updated_by,
		       s.id, s.name, s.type, s.version,
		       d.id, d.name, d.type, d.version
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
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=NOW(), updated_at=NOW(), updated_by_id=$2 WHERE id=$1`, m.tbl("job"))
	m.authMu.RLock()
	uid := m.userID
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

	q := fmt.Sprintf(`UPDATE %s SET active=$1, updated_by_id=$2, updated_at=NOW() WHERE id=$3`, m.tbl("job"))
	_, err := m.db.ExecContext(context.Background(), q, activate, m.userID, id)
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

// GetTaskLogs returns paginated log entries for a job task.
// For the direct service layer this is a stub — log files are written by the
// Temporal worker and live on the worker host. The TUI shows a message to the user.
func (m *Manager) GetTaskLogs(jobID int, taskID string, filePath string, cursor int64, limit int, direction string) (*TaskLogsResponse, error) {
	return &TaskLogsResponse{
		Logs: []LogEntry{{
			Level:   "info",
			Time:    time.Now().Format(time.RFC3339),
			Message: "Log streaming requires access to the Temporal worker host filesystem. Use the BFF API for remote log access.",
		}},
	}, nil
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
	q := fmt.Sprintf(`UPDATE %s SET name=$1, frequency=$2, updated_by_id=$3, updated_at=NOW() WHERE id=$4`, m.tbl("job"))
	_, err := m.db.ExecContext(context.Background(), q, name, frequency, m.userID, id)
	return err
}

// ClearDestination triggers a clear-destination workflow for a job.
//
// The BFF uses a dedicated Temporal workflow that:
//  1. Pauses the normal sync schedule.
//  2. Updates the schedule to use a "clear" execution request (job_type=clear).
//  3. Triggers the updated schedule.
//  4. After completion, the worker restores the normal sync schedule.
//
// In the direct-DB mode the TUI cannot replicate this multi-step orchestration
// without the full BFF service layer. We therefore return a clear error so the
// user is not misled into thinking a normal sync has cleared their destination.
// If you need clear-destination support, use the BFF API endpoint:
//   POST /api/v1/project/{projectid}/jobs/{id}/clear-destination
func (m *Manager) ClearDestination(id int) error {
	return fmt.Errorf(
		"clear-destination requires the BFF service layer and cannot be " +
			"executed from the direct-DB mode. Use the BFF API endpoint instead: " +
			"POST /api/v1/project/{projectid}/jobs/%d/clear-destination",
		id,
	)
}

// TestSource is not implemented in the direct service layer (requires Temporal worker).
func (m *Manager) TestSource(s EntityBase) (*TestConnectionResult, error) {
	return &TestConnectionResult{}, fmt.Errorf("connection testing requires the Temporal worker; not available in direct DB mode")
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

// DiscoverStreams returns streams for a source.
// In direct-DB mode it tries to extract streams from an existing job that uses
// the same source. If none exists, it returns an error advising the user.
func (m *Manager) DiscoverStreams(sourceID int) ([]StreamInfo, error) {
	q := fmt.Sprintf(
		`SELECT streams_config FROM %s WHERE source_id=$1 AND streams_config IS NOT NULL AND streams_config <> '' LIMIT 1`,
		m.tbl("job"))

	var rawConfig string
	err := m.db.QueryRowContext(context.Background(), q, sourceID).Scan(&rawConfig)
	if err == sql.ErrNoRows || rawConfig == "" {
		// No existing job → return a helpful error rather than silently empty
		return nil, fmt.Errorf("no stream catalogue found for this source; " +
			"run an initial sync via the BFF to populate the stream catalogue")
	}
	if err != nil {
		return nil, fmt.Errorf("discover streams: %w", err)
	}

	// streams_config may be a JSON array of objects; attempt to extract names
	var raw []map[string]interface{}
	if err := json.Unmarshal([]byte(rawConfig), &raw); err != nil {
		return nil, fmt.Errorf("parse streams config: %w", err)
	}

	seen := map[string]bool{}
	var streams []StreamInfo
	for _, obj := range raw {
		ns, _ := obj["namespace"].(string)
		name, _ := obj["name"].(string)
		if name == "" {
			name, _ = obj["stream_name"].(string)
		}
		if name == "" {
			continue
		}
		key := ns + "." + name
		if seen[key] {
			continue
		}
		seen[key] = true

		// Try to extract sync modes from the config; fall back to sensible defaults
		modes := []string{"full_refresh", "incremental", "cdc"}
		if sm, ok := obj["supported_sync_modes"].([]interface{}); ok {
			modes = nil
			for _, v := range sm {
				if s, ok := v.(string); ok {
					modes = append(modes, s)
				}
			}
		}
		// Cursor fields
		var cursors []string
		if cf, ok := obj["available_cursor_fields"].([]interface{}); ok {
			for _, v := range cf {
				if s, ok := v.(string); ok {
					cursors = append(cursors, s)
				}
			}
		}
		streams = append(streams, StreamInfo{
			Namespace:    ns,
			Name:         name,
			SyncModes:    modes,
			CursorFields: cursors,
		})
	}
	return streams, nil
}

// CreateJob inserts a new job record into the database.
func (m *Manager) CreateJob(name string, sourceID, destID int, frequency string, streams []StreamConfig) (*Job, error) {
	if frequency == "" {
		frequency = "0 * * * *" // every hour default
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
		m.projectID, m.userID,
	).Scan(&jobID)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return m.GetJob(jobID)
}
