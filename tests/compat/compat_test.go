//go:build e2e

// Package compat provides end-to-end compatibility tests for OLake TUI.
//
// These tests are designed to run against a real OLake PostgreSQL database.
// They are guarded by the `e2e` build tag and are skipped automatically when
// the OLAKE_DB_URL environment variable is not set.
//
// Usage:
//
//	OLAKE_DB_URL=postgres://olake:olake@localhost:5432/olake?sslmode=disable \
//	  go test -tags e2e ./tests/compat/ -v
package compat

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/datazip-inc/olake-tui/internal/service"
)

// ─── Setup ───────────────────────────────────────────────────────────────────

// newService creates a Manager connected to the real database.
// It skips the test if OLAKE_DB_URL is not set.
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

	svc, err := service.New(service.Config{
		DBURL:         dbURL,
		EncryptionKey: encKey,
		RunMode:       runMode,
		ProjectID:     "123",
		TemporalHost:  "localhost:7233",
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

// uniqueName returns a test-scoped unique name to avoid collisions between runs.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-test-%d", prefix, time.Now().UnixNano())
}

// minimalPostgresConfig returns a JSON connector config for testing.
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

// ─── Scenario 1: Schema validation ───────────────────────────────────────────

func TestE2E_SchemaValidation(t *testing.T) {
	svc := newService(t)

	if err := svc.ValidateSchema(); err != nil {
		t.Errorf("schema validation failed: %v", err)
	}
}

func TestE2E_GetCompatibleVersion(t *testing.T) {
	svc := newService(t)

	v := svc.GetCompatibleVersion()
	if v == "" {
		t.Error("GetCompatibleVersion returned empty string")
	}
	if !strings.Contains(v, "olake") {
		t.Errorf("version string should mention olake, got %q", v)
	}
}

// ─── Scenario 2: Source CRUD ─────────────────────────────────────────────────

func TestE2E_CreateAndVerifySource(t *testing.T) {
	svc := newService(t)

	name := uniqueName("src")
	base := service.EntityBase{
		Name:    name,
		Type:    "postgres",
		Version: "1.0.0",
		Config:  minimalPostgresConfig(),
	}

	created, err := svc.CreateSource(base)
	if err != nil {
		t.Fatalf("CreateSource failed: %v", err)
	}
	if created.Name != name {
		t.Errorf("name mismatch: want %q, got %q", name, created.Name)
	}

	// Verify it appears in the list.
	sources, err := svc.ListSources()
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	found := false
	var foundID int
	for _, s := range sources {
		if s.Name == name {
			found = true
			foundID = s.ID
			break
		}
	}
	if !found {
		t.Fatalf("created source %q not found in ListSources", name)
	}

	// Verify GetSource.
	src, err := svc.GetSource(foundID)
	if err != nil {
		t.Fatalf("GetSource(%d) failed: %v", foundID, err)
	}
	if src.Name != name {
		t.Errorf("GetSource name mismatch: want %q, got %q", name, src.Name)
	}

	// Cleanup — delete the source.
	t.Cleanup(func() {
		_ = svc.DeleteSource(foundID)
	})
}

func TestE2E_DeleteSourceSoftDeletes(t *testing.T) {
	svc := newService(t)

	name := uniqueName("src-del")
	_, err := svc.CreateSource(service.EntityBase{
		Name:    name,
		Type:    "postgres",
		Version: "1.0.0",
		Config:  minimalPostgresConfig(),
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	// Find and delete.
	sources, _ := svc.ListSources()
	var id int
	for _, s := range sources {
		if s.Name == name {
			id = s.ID
			break
		}
	}
	if id == 0 {
		t.Fatal("source not found after creation")
	}

	if err := svc.DeleteSource(id); err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}

	// Should no longer appear in list (soft-deleted).
	sources2, _ := svc.ListSources()
	for _, s := range sources2 {
		if s.ID == id {
			t.Error("soft-deleted source still appears in ListSources")
		}
	}

	// GetSource should also fail.
	_, err = svc.GetSource(id)
	if err == nil {
		t.Error("GetSource on soft-deleted source should return error")
	}
}

// ─── Scenario 3: Destination CRUD ────────────────────────────────────────────

func TestE2E_CreateAndVerifyDestination(t *testing.T) {
	svc := newService(t)

	name := uniqueName("dst")
	cfg := `{"bucket":"test-bucket","region":"us-east-1"}`
	base := service.EntityBase{
		Name:    name,
		Type:    "s3",
		Version: "1.0.0",
		Config:  cfg,
	}

	created, err := svc.CreateDestination(base)
	if err != nil {
		t.Fatalf("CreateDestination failed: %v", err)
	}
	if created.Name != name {
		t.Errorf("name mismatch: want %q, got %q", name, created.Name)
	}

	dests, err := svc.ListDestinations()
	if err != nil {
		t.Fatalf("ListDestinations failed: %v", err)
	}

	var foundID int
	for _, d := range dests {
		if d.Name == name {
			foundID = d.ID
			break
		}
	}
	if foundID == 0 {
		t.Fatalf("created destination %q not found in ListDestinations", name)
	}

	t.Cleanup(func() {
		_ = svc.DeleteDestination(foundID)
	})
}

// ─── Scenario 4: Job CRUD ─────────────────────────────────────────────────────

func TestE2E_CreateAndVerifyJob(t *testing.T) {
	svc := newService(t)

	// Create source and destination first.
	srcName := uniqueName("src-job")
	_, err := svc.CreateSource(service.EntityBase{
		Name: srcName, Type: "postgres", Version: "1.0.0",
		Config: minimalPostgresConfig(),
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	dstName := uniqueName("dst-job")
	_, err = svc.CreateDestination(service.EntityBase{
		Name: dstName, Type: "s3", Version: "1.0.0",
		Config: `{"bucket":"test"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}

	// Resolve IDs.
	sources, _ := svc.ListSources()
	dests, _ := svc.ListDestinations()
	var srcID, dstID int
	for _, s := range sources {
		if s.Name == srcName {
			srcID = s.ID
		}
	}
	for _, d := range dests {
		if d.Name == dstName {
			dstID = d.ID
		}
	}
	if srcID == 0 || dstID == 0 {
		t.Fatal("could not resolve source/destination IDs")
	}

	jobName := uniqueName("job")
	streams := []service.StreamConfig{
		{Namespace: "public", Name: "users", SyncMode: "full_refresh", Selected: true},
	}

	job, err := svc.CreateJob(jobName, srcID, dstID, "0 * * * *", streams)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if job.Name != jobName {
		t.Errorf("job name mismatch: want %q, got %q", jobName, job.Name)
	}
	if job.ID == 0 {
		t.Error("created job should have a non-zero ID")
	}

	// Verify in list.
	jobs, err := svc.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	found := false
	for _, j := range jobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created job not found in ListJobs")
	}

	// Cleanup.
	t.Cleanup(func() {
		_ = svc.DeleteJob(job.ID)
		_ = svc.DeleteSource(srcID)
		_ = svc.DeleteDestination(dstID)
	})
}

// ─── Scenario 5: List counts ──────────────────────────────────────────────────

func TestE2E_ListCounts(t *testing.T) {
	svc := newService(t)

	sources, err := svc.ListSources()
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	t.Logf("sources: %d", len(sources))

	dests, err := svc.ListDestinations()
	if err != nil {
		t.Fatalf("ListDestinations: %v", err)
	}
	t.Logf("destinations: %d", len(dests))

	jobs, err := svc.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	t.Logf("jobs: %d", len(jobs))

	// All lists must be non-nil (even if empty).
	if sources == nil {
		t.Error("ListSources returned nil")
	}
	if dests == nil {
		t.Error("ListDestinations returned nil")
	}
	if jobs == nil {
		t.Error("ListJobs returned nil")
	}
}

// ─── Scenario 6: Encryption round-trip ───────────────────────────────────────

func TestE2E_EncryptionRoundTrip(t *testing.T) {
	svc := newService(t)

	encKey := os.Getenv("OLAKE_SECRET_KEY")
	if encKey == "" {
		t.Skip("OLAKE_SECRET_KEY not set — skipping encryption round-trip test")
	}

	sensitiveConfig := `{"host":"db","port":5432,"password":"super-secret-password"}`
	name := uniqueName("src-enc")

	_, err := svc.CreateSource(service.EntityBase{
		Name: name, Type: "postgres", Version: "1.0.0",
		Config: sensitiveConfig,
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	sources, _ := svc.ListSources()
	var srcID int
	for _, s := range sources {
		if s.Name == name {
			srcID = s.ID
			// The config returned by the service should be the plaintext (decrypted).
			if s.Config != sensitiveConfig {
				t.Errorf("decrypted config mismatch:\n  want: %q\n  got:  %q", sensitiveConfig, s.Config)
			}
			break
		}
	}
	if srcID == 0 {
		t.Fatal("source not found")
	}

	t.Cleanup(func() {
		_ = svc.DeleteSource(srcID)
	})
}

// ─── Scenario 7: Settings ─────────────────────────────────────────────────────

func TestE2E_GetAndUpdateSettings(t *testing.T) {
	svc := newService(t)

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	t.Logf("settings: %+v", settings)

	// Update webhook URL.
	settings.WebhookAlertURL = "https://example.com/webhook-test"
	if err := svc.UpdateSettings(*settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	// Read back.
	settings2, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after update: %v", err)
	}
	if settings2.WebhookAlertURL != settings.WebhookAlertURL {
		t.Errorf("webhook URL not persisted: want %q, got %q",
			settings.WebhookAlertURL, settings2.WebhookAlertURL)
	}
}
