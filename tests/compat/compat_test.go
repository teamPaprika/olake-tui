//go:build e2e

// Package compat provides end-to-end compatibility tests for OLake TUI.
//
// These tests run against a real OLake PostgreSQL database and optionally
// Temporal, verifying that olake-tui reads/writes data in a format fully
// compatible with the official OLake BFF server.
//
// Usage:
//
//	OLAKE_DB_URL=postgres://olake:olake@localhost:5432/olake?sslmode=disable \
//	  go test -tags e2e ./tests/compat/ -v
//
// With encryption + Temporal:
//
//	OLAKE_DB_URL=postgres://olake:olake@localhost:5432/olake?sslmode=disable \
//	OLAKE_SECRET_KEY=mysecretkey \
//	TEMPORAL_ADDRESS=localhost:7233 \
//	  go test -tags e2e ./tests/compat/ -v
package compat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Setup ───────────────────────────────────────────────────────────────────

func newService(t *testing.T) *service.Manager {
	t.Helper()

	dbURL := os.Getenv("OLAKE_DB_URL")
	if dbURL == "" {
		t.Skip("OLAKE_DB_URL not set — skipping E2E test")
	}

	encKey := os.Getenv("OLAKE_SECRET_KEY")
	runMode := os.Getenv("OLAKE_RUN_MODE")
	if runMode == "" {
		runMode = "dev"
	}

	temporalHost := os.Getenv("TEMPORAL_ADDRESS")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	svc, err := service.New(service.Config{
		DBURL:         dbURL,
		EncryptionKey: encKey,
		RunMode:       runMode,
		ProjectID:     "123",
		TemporalHost:  temporalHost,
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

// rawDB opens a raw sql.DB for cross-checking TUI writes with BFF expectations.
func rawDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("OLAKE_DB_URL")
	if dbURL == "" {
		t.Skip("OLAKE_DB_URL not set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("raw db open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-test-%d", prefix, time.Now().UnixNano())
}

func minimalPostgresConfig() string {
	cfg := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "testdb",
		"username": "testuser",
		"password": "testpass",
	}
	data, _ := json.Marshal(cfg)
	return string(data)
}

func runMode() string {
	rm := os.Getenv("OLAKE_RUN_MODE")
	if rm == "" {
		return "dev"
	}
	return rm
}

func tbl(entity string) string {
	return fmt.Sprintf(`"olake-%s-%s"`, runMode(), entity)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 1. SCHEMA & VERSION
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_SchemaValidation(t *testing.T) {
	svc := newService(t)
	if err := svc.ValidateSchema(); err != nil {
		t.Errorf("schema validation failed: %v", err)
	}
}

func TestE2E_GetCompatibleVersion(t *testing.T) {
	svc := newService(t)
	v := svc.GetCompatibleVersion()
	if v == "" || !strings.Contains(v, "olake") {
		t.Errorf("unexpected version string: %q", v)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 2. AUTH — login against real BFF user table
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_LoginInvalidCredentials(t *testing.T) {
	svc := newService(t)
	err := svc.Login("nonexistent-user-xyz", "badpassword")
	if err == nil {
		t.Error("Login with bad credentials should fail")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 3. SOURCE CRUD + BFF cross-compatibility
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_SourceCRUD(t *testing.T) {
	svc := newService(t)

	// Must be logged in for created_by_id
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))

	name := uniqueName("src")
	base := service.EntityBase{
		Name:    name,
		Type:    "postgres",
		Version: "1.0.0",
		Config:  minimalPostgresConfig(),
	}

	// CREATE
	created, err := svc.CreateSource(base)
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	if created.Name != name {
		t.Errorf("name mismatch: want %q, got %q", name, created.Name)
	}

	// LIST — must appear
	sources, _ := svc.ListSources()
	var srcID int
	for _, s := range sources {
		if s.Name == name {
			srcID = s.ID
			break
		}
	}
	if srcID == 0 {
		t.Fatalf("created source %q not in ListSources", name)
	}

	// GET
	src, err := svc.GetSource(srcID)
	if err != nil {
		t.Fatalf("GetSource(%d): %v", srcID, err)
	}
	if src.Type != "postgres" {
		t.Errorf("type mismatch: %q", src.Type)
	}

	// UPDATE
	updated, err := svc.UpdateSource(srcID, service.EntityBase{
		Name: name + "-updated", Type: "postgres", Version: "1.0.0",
		Config: minimalPostgresConfig(),
	})
	if err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}
	if !strings.HasSuffix(updated.Name, "-updated") {
		t.Errorf("update didn't change name: %q", updated.Name)
	}

	// DELETE (soft)
	if err := svc.DeleteSource(srcID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	// After delete — should not appear
	sources2, _ := svc.ListSources()
	for _, s := range sources2 {
		if s.ID == srcID {
			t.Error("soft-deleted source still in list")
		}
	}
}

// TestE2E_SourceEncryptionBFFCompat verifies that config encrypted by TUI
// is stored in the same format as BFF (JSON-quoted base64 of AES-256-GCM).
func TestE2E_SourceEncryptionBFFCompat(t *testing.T) {
	encKey := os.Getenv("OLAKE_SECRET_KEY")
	if encKey == "" {
		t.Skip("OLAKE_SECRET_KEY not set")
	}

	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))
	db := rawDB(t)

	name := uniqueName("src-enc")
	plainConfig := `{"host":"db.example.com","port":5432,"password":"s3cret!"}`
	_, err := svc.CreateSource(service.EntityBase{
		Name: name, Type: "postgres", Version: "1.0.0", Config: plainConfig,
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	// Read raw encrypted value from DB (as BFF would see it)
	var rawConfig string
	q := fmt.Sprintf(`SELECT config FROM %s WHERE name=$1 AND deleted_at IS NULL`, tbl("source"))
	if err := db.QueryRow(q, name).Scan(&rawConfig); err != nil {
		t.Fatalf("raw query: %v", err)
	}

	// BFF format: JSON-quoted base64 string — must start with a quote
	if !strings.HasPrefix(rawConfig, `"`) {
		t.Errorf("encrypted config should be JSON-quoted base64, got prefix: %q", rawConfig[:20])
	}

	// TUI decrypts it back correctly
	src, _ := svc.GetSource(findSourceID(t, svc, name))
	if src.Config != plainConfig {
		t.Errorf("decrypt mismatch:\n  want: %s\n  got:  %s", plainConfig, src.Config)
	}

	// Cleanup
	t.Cleanup(func() { _ = svc.DeleteSource(src.ID) })
}

// TestE2E_SourceDeleteBlockedByJob verifies you can't delete a source with active jobs.
func TestE2E_SourceDeleteBlockedByJob(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))

	srcName := uniqueName("src-block")
	_, _ = svc.CreateSource(service.EntityBase{
		Name: srcName, Type: "postgres", Version: "1.0.0", Config: minimalPostgresConfig(),
	})
	dstName := uniqueName("dst-block")
	_, _ = svc.CreateDestination(service.EntityBase{
		Name: dstName, Type: "iceberg", Version: "1.0.0", Config: `{"warehouse":"s3://test"}`,
	})

	srcID := findSourceID(t, svc, srcName)
	dstID := findDestID(t, svc, dstName)

	job, err := svc.CreateJob(uniqueName("job-block"), srcID, dstID, "0 * * * *", nil)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	// Should fail — source has a job
	err = svc.DeleteSource(srcID)
	if err == nil {
		t.Error("DeleteSource should fail when jobs reference it")
	}

	t.Cleanup(func() {
		_ = svc.DeleteJob(job.ID)
		_ = svc.DeleteSource(srcID)
		_ = svc.DeleteDestination(dstID)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// 4. DESTINATION CRUD
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_DestinationCRUD(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))

	name := uniqueName("dst")
	base := service.EntityBase{
		Name: name, Type: "iceberg", Version: "1.0.0",
		Config: `{"catalog_type":"hive","warehouse":"s3://test","uri":"http://localhost:9083"}`,
	}

	_, err := svc.CreateDestination(base)
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}

	dstID := findDestID(t, svc, name)

	// Update
	_, err = svc.UpdateDestination(dstID, service.EntityBase{
		Name: name + "-v2", Type: "iceberg", Version: "1.0.0",
		Config: base.Config,
	})
	if err != nil {
		t.Fatalf("UpdateDestination: %v", err)
	}

	// Delete
	if err := svc.DeleteDestination(dstID); err != nil {
		t.Fatalf("DeleteDestination: %v", err)
	}

	// Gone from list
	dests, _ := svc.ListDestinations()
	for _, d := range dests {
		if d.ID == dstID {
			t.Error("deleted dest still in list")
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// 5. JOB lifecycle — create, update, activate, delete
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_JobLifecycle(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))

	srcName := uniqueName("src-jlc")
	dstName := uniqueName("dst-jlc")
	_, _ = svc.CreateSource(service.EntityBase{
		Name: srcName, Type: "postgres", Version: "1.0.0", Config: minimalPostgresConfig(),
	})
	_, _ = svc.CreateDestination(service.EntityBase{
		Name: dstName, Type: "iceberg", Version: "1.0.0", Config: `{"warehouse":"s3://test"}`,
	})

	srcID := findSourceID(t, svc, srcName)
	dstID := findDestID(t, svc, dstName)

	streams := []service.StreamConfig{
		{Namespace: "public", Name: "users", SyncMode: "full_refresh", Selected: true},
		{Namespace: "public", Name: "orders", SyncMode: "incremental", CursorField: "updated_at", Selected: true},
	}

	// CREATE
	jobName := uniqueName("job-lc")
	job, err := svc.CreateJob(jobName, srcID, dstID, "0 * * * *", streams)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("job ID should be non-zero")
	}

	// VERIFY in list
	jobs, _ := svc.ListJobs()
	found := false
	for _, j := range jobs {
		if j.ID == job.ID {
			found = true
			if j.Source.ID != srcID {
				t.Errorf("source ID mismatch: want %d, got %d", srcID, j.Source.ID)
			}
			if j.Destination.ID != dstID {
				t.Errorf("dest ID mismatch: want %d, got %d", dstID, j.Destination.ID)
			}
			break
		}
	}
	if !found {
		t.Error("job not found in ListJobs")
	}

	// UPDATE META
	if err := svc.UpdateJobMeta(job.ID, jobName+"-renamed", "*/30 * * * *"); err != nil {
		t.Fatalf("UpdateJobMeta: %v", err)
	}
	updated, _ := svc.GetJob(job.ID)
	if updated.Name != jobName+"-renamed" {
		t.Errorf("name not updated: %q", updated.Name)
	}
	if updated.Frequency != "*/30 * * * *" {
		t.Errorf("frequency not updated: %q", updated.Frequency)
	}

	// ACTIVATE (pause)
	if err := svc.ActivateJob(job.ID, false); err != nil {
		t.Fatalf("ActivateJob(false): %v", err)
	}
	paused, _ := svc.GetJob(job.ID)
	if paused.Activate {
		t.Error("job should be paused")
	}

	// ACTIVATE (resume)
	if err := svc.ActivateJob(job.ID, true); err != nil {
		t.Fatalf("ActivateJob(true): %v", err)
	}

	// DELETE
	if err := svc.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
	jobs2, _ := svc.ListJobs()
	for _, j := range jobs2 {
		if j.ID == job.ID {
			t.Error("deleted job still in list")
		}
	}

	t.Cleanup(func() {
		_ = svc.DeleteSource(srcID)
		_ = svc.DeleteDestination(dstID)
	})
}

// TestE2E_JobNameUniqueness verifies duplicate names are rejected.
func TestE2E_JobNameUniqueness(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))

	srcName := uniqueName("src-uniq")
	dstName := uniqueName("dst-uniq")
	_, _ = svc.CreateSource(service.EntityBase{
		Name: srcName, Type: "postgres", Version: "1.0.0", Config: minimalPostgresConfig(),
	})
	_, _ = svc.CreateDestination(service.EntityBase{
		Name: dstName, Type: "iceberg", Version: "1.0.0", Config: `{"warehouse":"s3://test"}`,
	})
	srcID := findSourceID(t, svc, srcName)
	dstID := findDestID(t, svc, dstName)

	jobName := uniqueName("job-dup")
	job, err := svc.CreateJob(jobName, srcID, dstID, "0 * * * *", nil)
	if err != nil {
		t.Fatalf("first CreateJob: %v", err)
	}

	// Second create with same name should fail
	_, err = svc.CreateJob(jobName, srcID, dstID, "0 * * * *", nil)
	if err == nil {
		t.Error("duplicate job name should be rejected")
	}

	t.Cleanup(func() {
		_ = svc.DeleteJob(job.ID)
		_ = svc.DeleteSource(srcID)
		_ = svc.DeleteDestination(dstID)
	})
}

// TestE2E_JobStreamsConfigPersisted verifies streams_config JSON is stored and retrievable.
func TestE2E_JobStreamsConfigPersisted(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))
	db := rawDB(t)

	srcName := uniqueName("src-strm")
	dstName := uniqueName("dst-strm")
	_, _ = svc.CreateSource(service.EntityBase{
		Name: srcName, Type: "postgres", Version: "1.0.0", Config: minimalPostgresConfig(),
	})
	_, _ = svc.CreateDestination(service.EntityBase{
		Name: dstName, Type: "iceberg", Version: "1.0.0", Config: `{"warehouse":"s3://test"}`,
	})
	srcID := findSourceID(t, svc, srcName)
	dstID := findDestID(t, svc, dstName)

	streams := []service.StreamConfig{
		{Namespace: "public", Name: "users", SyncMode: "incremental", CursorField: "id", Normalize: true, Selected: true},
		{Namespace: "public", Name: "events", SyncMode: "full_refresh", Selected: true},
	}

	job, err := svc.CreateJob(uniqueName("job-strm"), srcID, dstID, "0 * * * *", streams)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	// Read raw streams_config from DB — this is what BFF worker consumes
	var rawStreams string
	q := fmt.Sprintf(`SELECT streams_config FROM %s WHERE id=$1`, tbl("job"))
	if err := db.QueryRow(q, job.ID).Scan(&rawStreams); err != nil {
		t.Fatalf("raw query: %v", err)
	}

	// Must be valid JSON array
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(rawStreams), &parsed); err != nil {
		t.Fatalf("streams_config is not valid JSON: %v\nraw: %s", err, rawStreams)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 streams, got %d", len(parsed))
	}

	// Verify first stream has expected fields
	if parsed[0]["namespace"] != "public" || parsed[0]["name"] != "users" {
		t.Errorf("first stream unexpected: %v", parsed[0])
	}

	t.Cleanup(func() {
		_ = svc.DeleteJob(job.ID)
		_ = svc.DeleteSource(srcID)
		_ = svc.DeleteDestination(dstID)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// 6. SETTINGS
// ═══════════════════════════════════════════════════════════════════════════════

func TestE2E_Settings(t *testing.T) {
	svc := newService(t)

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	original := settings.WebhookAlertURL
	settings.WebhookAlertURL = "https://hooks.example.com/test-" + fmt.Sprint(time.Now().UnixNano())
	if err := svc.UpdateSettings(*settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	settings2, _ := svc.GetSettings()
	if settings2.WebhookAlertURL != settings.WebhookAlertURL {
		t.Errorf("webhook not persisted: want %q, got %q", settings.WebhookAlertURL, settings2.WebhookAlertURL)
	}

	// Restore
	t.Cleanup(func() {
		settings.WebhookAlertURL = original
		_ = svc.UpdateSettings(*settings)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7. CROSS-COMPATIBILITY — TUI writes readable by BFF, and vice versa
// ═══════════════════════════════════════════════════════════════════════════════

// TestE2E_SoftDeleteConsistency verifies that TUI soft-delete (deleted_at=NOW())
// matches BFF Beego ORM soft-delete semantics.
func TestE2E_SoftDeleteConsistency(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))
	db := rawDB(t)

	name := uniqueName("src-sd")
	_, _ = svc.CreateSource(service.EntityBase{
		Name: name, Type: "postgres", Version: "1.0.0", Config: minimalPostgresConfig(),
	})
	srcID := findSourceID(t, svc, name)
	_ = svc.DeleteSource(srcID)

	// Raw DB: deleted_at should be NOT NULL
	var deletedAt sql.NullTime
	q := fmt.Sprintf(`SELECT deleted_at FROM %s WHERE id=$1`, tbl("source"))
	if err := db.QueryRow(q, srcID).Scan(&deletedAt); err != nil {
		t.Fatalf("raw query: %v", err)
	}
	if !deletedAt.Valid {
		t.Error("soft-deleted source should have non-null deleted_at")
	}
}

// TestE2E_TableNamingConvention verifies TUI uses the same table name prefix as BFF.
func TestE2E_TableNamingConvention(t *testing.T) {
	db := rawDB(t)
	rm := runMode()

	expected := []string{"source", "destination", "job", "user", "project-settings"}
	for _, entity := range expected {
		tableName := fmt.Sprintf("olake-%s-%s", rm, entity)
		var exists bool
		err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`, tableName).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", tableName, err)
		}
		if !exists {
			t.Errorf("expected table %q not found", tableName)
		}
	}
}

// TestE2E_PortStoredAsNumber verifies config port is stored as JSON number (BFF compat).
func TestE2E_PortStoredAsNumber(t *testing.T) {
	svc := newService(t)
	_ = svc.Login("admin", os.Getenv("OLAKE_ADMIN_PASSWORD"))
	db := rawDB(t)

	encKey := os.Getenv("OLAKE_SECRET_KEY")
	if encKey == "" {
		t.Skip("OLAKE_SECRET_KEY needed to verify encrypted config format")
	}

	name := uniqueName("src-port")
	cfg := `{"host":"localhost","port":5432,"username":"test","password":"test","database":"test"}`
	_, _ = svc.CreateSource(service.EntityBase{
		Name: name, Type: "postgres", Version: "1.0.0", Config: cfg,
	})
	srcID := findSourceID(t, svc, name)

	// TUI should decrypt and return the config with port as number
	src, _ := svc.GetSource(srcID)
	var parsed map[string]interface{}
	_ = json.Unmarshal([]byte(src.Config), &parsed)

	// port should be float64 (JSON number), not string
	port, ok := parsed["port"].(float64)
	if !ok {
		t.Errorf("port should be JSON number, got %T: %v", parsed["port"], parsed["port"])
	} else if port != 5432 {
		t.Errorf("port value mismatch: want 5432, got %v", port)
	}

	// Also verify the raw encrypted value in DB is JSON-quoted base64
	var rawConfig string
	q := fmt.Sprintf(`SELECT config FROM %s WHERE id=$1`, tbl("source"))
	_ = db.QueryRow(q, srcID).Scan(&rawConfig)
	if !strings.HasPrefix(rawConfig, `"`) {
		t.Errorf("encrypted config not in BFF format (JSON-quoted base64)")
	}

	t.Cleanup(func() { _ = svc.DeleteSource(srcID) })
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func findSourceID(t *testing.T, svc *service.Manager, name string) int {
	t.Helper()
	sources, _ := svc.ListSources()
	for _, s := range sources {
		if s.Name == name {
			return s.ID
		}
	}
	t.Fatalf("source %q not found", name)
	return 0
}

func findDestID(t *testing.T, svc *service.Manager, name string) int {
	t.Helper()
	dests, _ := svc.ListDestinations()
	for _, d := range dests {
		if d.Name == name {
			return d.ID
		}
	}
	t.Fatalf("destination %q not found", name)
	return 0
}
