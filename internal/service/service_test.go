package service

import (
	"encoding/json"
	"strings"
	"testing"
)

// ─── Encryption tests ─────────────────────────────────────────────────────────

func newManagerWithKey(key string) *Manager {
	return &Manager{
		encryptionKey: key,
		runMode:       "dev",
		projectID:     "123",
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	m := newManagerWithKey("supersecretkey")
	plaintext := `{"host":"localhost","port":5432,"user":"admin","password":"secret123"}`

	encrypted, err := m.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("encrypt returned plaintext unchanged — encryption is not working")
	}

	decrypted, err := m.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("round-trip mismatch:\n  want: %q\n  got:  %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptRoundTrip_DifferentNonces(t *testing.T) {
	m := newManagerWithKey("anotherkey42")
	plaintext := "hello world"

	enc1, err := m.encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := m.encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	// AES-GCM uses a random nonce, so the same plaintext must yield different ciphertexts.
	if enc1 == enc2 {
		t.Error("two encryptions of the same plaintext produced identical ciphertext — nonce is not random")
	}

	d1, _ := m.decrypt(enc1)
	d2, _ := m.decrypt(enc2)
	if d1 != plaintext || d2 != plaintext {
		t.Errorf("decrypted values don't match: %q, %q", d1, d2)
	}
}

func TestEncryptNoKey(t *testing.T) {
	m := newManagerWithKey("") // no encryption key
	plaintext := "config data"

	encrypted, err := m.encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	// Without a key, encrypt should return the original plaintext unchanged.
	if encrypted != plaintext {
		t.Errorf("with no key, encrypt should be identity; got %q", encrypted)
	}

	decrypted, err := m.decrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plaintext {
		t.Errorf("with no key, decrypt should be identity; got %q", decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	m1 := newManagerWithKey("keyA")
	m2 := newManagerWithKey("keyB")

	encrypted, err := m1.encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}

	_, err = m2.decrypt(encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong key, got nil")
	}
}

func TestDecryptEmptyString(t *testing.T) {
	m := newManagerWithKey("somekey")
	result, err := m.decrypt("")
	if err != nil {
		t.Fatalf("unexpected error decrypting empty string: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for empty input, got %q", result)
	}
}

func TestEncryptEmptyString(t *testing.T) {
	m := newManagerWithKey("somekey")
	result, err := m.encrypt("")
	if err != nil {
		t.Fatalf("unexpected error encrypting empty string: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for empty input, got %q", result)
	}
}

// ─── Config JSON marshaling tests ─────────────────────────────────────────────

func TestEntityBaseJSONMarshal(t *testing.T) {
	e := EntityBase{
		Name:    "my-source",
		Type:    "postgres",
		Version: "1.0.0",
		Config:  `{"host":"db","port":5432}`,
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded EntityBase
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Name != e.Name || decoded.Type != e.Type || decoded.Version != e.Version || decoded.Config != e.Config {
		t.Errorf("round-trip mismatch: want %+v, got %+v", e, decoded)
	}
}

func TestStreamConfigJSONMarshal(t *testing.T) {
	configs := []StreamConfig{
		{Namespace: "public", Name: "users", SyncMode: "cdc", CursorField: "id", Normalize: true, Selected: true},
		{Namespace: "", Name: "orders", SyncMode: "full_refresh", Selected: false},
	}

	data, err := json.Marshal(configs)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded []StreamConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(decoded) != len(configs) {
		t.Fatalf("length mismatch: want %d, got %d", len(configs), len(decoded))
	}
	for i, sc := range configs {
		if decoded[i].Name != sc.Name || decoded[i].SyncMode != sc.SyncMode {
			t.Errorf("[%d] mismatch: want %+v, got %+v", i, sc, decoded[i])
		}
	}
}

func TestAdvancedSettingsJSON(t *testing.T) {
	n := 4
	a := AdvancedSettings{MaxDiscoverThreads: &n}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}

	var got AdvancedSettings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.MaxDiscoverThreads == nil || *got.MaxDiscoverThreads != n {
		t.Errorf("expected MaxDiscoverThreads=%d, got %v", n, got.MaxDiscoverThreads)
	}
}

// ─── Schema validation (via mock) ────────────────────────────────────────────

func TestGetCompatibleVersion(t *testing.T) {
	m := newManagerWithKey("")
	v := m.GetCompatibleVersion()
	if v == "" {
		t.Error("GetCompatibleVersion returned empty string")
	}
	if !strings.Contains(v, "olake") {
		t.Errorf("expected version string to mention olake, got %q", v)
	}
}

// ─── Table naming ─────────────────────────────────────────────────────────────

func TestTblName(t *testing.T) {
	tests := []struct {
		mode   string
		entity string
		want   string
	}{
		{"dev", "source", `"olake-dev-source"`},
		{"prod", "destination", `"olake-prod-destination"`},
		{"staging", "job", `"olake-staging-job"`},
		{"dev", "user", `"olake-dev-user"`},
		{"dev", "project-settings", `"olake-dev-project-settings"`},
	}

	for _, tt := range tests {
		m := &Manager{runMode: tt.mode}
		got := m.tbl(tt.entity)
		if got != tt.want {
			t.Errorf("tbl(%q) with mode=%q: want %q, got %q", tt.entity, tt.mode, tt.want, got)
		}
	}
}

// ─── MockService tests ────────────────────────────────────────────────────────

func TestMockServiceLogin(t *testing.T) {
	m := NewMockService()

	if err := m.Login("admin", "pass"); err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if !m.IsAuthenticated() {
		t.Error("expected authenticated=true after login")
	}
	if m.Username() != "admin" {
		t.Errorf("username mismatch: want admin, got %q", m.Username())
	}
}

func TestMockServiceLoginError(t *testing.T) {
	m := NewMockService()
	m.LoginErr = ErrBadCredentials

	err := m.Login("admin", "wrong")
	if err == nil {
		t.Fatal("expected login error, got nil")
	}
	if m.IsAuthenticated() {
		t.Error("should not be authenticated after failed login")
	}
}

var ErrBadCredentials = errBadCredentials{}

type errBadCredentials struct{}

func (e errBadCredentials) Error() string { return "invalid credentials" }

func TestMockServiceCRUD(t *testing.T) {
	m := NewMockService()
	m.Sources = []Source{
		{ID: 1, Name: "pg-prod", Type: "postgres"},
		{ID: 2, Name: "mongo-dev", Type: "mongodb"},
	}

	sources, err := m.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Errorf("want 2 sources, got %d", len(sources))
	}

	src, err := m.GetSource(1)
	if err != nil {
		t.Fatal(err)
	}
	if src.Name != "pg-prod" {
		t.Errorf("want pg-prod, got %q", src.Name)
	}

	_, err = m.GetSource(99)
	if err == nil {
		t.Error("expected error for non-existent source")
	}
}

func TestMockServiceCallCounting(t *testing.T) {
	m := NewMockService()
	_, _ = m.ListSources()
	_, _ = m.ListSources()
	_, _ = m.ListDestinations()

	if m.Calls["ListSources"] != 2 {
		t.Errorf("want 2 ListSources calls, got %d", m.Calls["ListSources"])
	}
	if m.Calls["ListDestinations"] != 1 {
		t.Errorf("want 1 ListDestinations call, got %d", m.Calls["ListDestinations"])
	}
}

func TestMockServiceCreateAndDelete(t *testing.T) {
	m := NewMockService()

	_, err := m.CreateSource(EntityBase{Name: "new-src", Type: "postgres", Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Sources) != 1 {
		t.Errorf("want 1 source after create, got %d", len(m.Sources))
	}

	id := m.Sources[0].ID
	if err := m.DeleteSource(id); err != nil {
		t.Fatal(err)
	}
	if len(m.Sources) != 0 {
		t.Error("source still present after delete")
	}
}

func TestMockServiceJobCreation(t *testing.T) {
	m := NewMockService()
	m.Sources = []Source{{ID: 1, Name: "src", Type: "postgres"}}
	m.Destinations = []Destination{{ID: 2, Name: "dst", Type: "iceberg"}}

	job, err := m.CreateJob("my-job", 1, 2, "0 * * * *", nil)
	if err != nil {
		t.Fatal(err)
	}
	if job.Name != "my-job" {
		t.Errorf("job name mismatch: want my-job, got %q", job.Name)
	}
	if job.Source.Name != "src" {
		t.Errorf("job source mismatch: want src, got %q", job.Source.Name)
	}
}

func TestMockServiceValidateSchema(t *testing.T) {
	m := NewMockService()

	// Happy path
	if err := m.ValidateSchema(); err != nil {
		t.Errorf("unexpected schema error: %v", err)
	}

	// Injected error
	m.ValidateSchemaErr = errBadCredentials{}
	if err := m.ValidateSchema(); err == nil {
		t.Error("expected schema validation error, got nil")
	}
}

// ─── IsNameUnique tests ───────────────────────────────────────────────────────

func TestMockIsNameUnique_Default(t *testing.T) {
	m := NewMockService()
	ok, err := m.IsNameUnique("job", "any-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("default mock should return true (unique)")
	}
}

func TestMockIsNameUnique_ErrorInjection(t *testing.T) {
	m := NewMockService()
	m.IsNameUniqueErr = errBadCredentials{}
	_, err := m.IsNameUnique("source", "test")
	if err == nil {
		t.Error("expected error from injected IsNameUniqueErr")
	}
}

func TestMockIsNameUnique_CallCounting(t *testing.T) {
	m := NewMockService()
	_, _ = m.IsNameUnique("job", "a")
	_, _ = m.IsNameUnique("source", "b")
	if m.Calls["IsNameUnique"] != 2 {
		t.Errorf("want 2 calls, got %d", m.Calls["IsNameUnique"])
	}
}

// ─── UpdateJobFull tests ──────────────────────────────────────────────────────

func TestMockUpdateJobFull_Success(t *testing.T) {
	m := NewMockService()
	m.Jobs = []Job{{ID: 1, Name: "old", Frequency: "0 * * * *", Activate: true}}

	err := m.UpdateJobFull(1, "new-name", 10, 20, "*/5 * * * *",
		[]StreamConfig{{Name: "users", Selected: true}}, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mock updated the job
	job, _ := m.GetJob(1)
	if job.Name != "new-name" {
		t.Errorf("name not updated: %q", job.Name)
	}
	if job.Frequency != "*/5 * * * *" {
		t.Errorf("frequency not updated: %q", job.Frequency)
	}
	if job.Activate {
		t.Error("activate should be false")
	}
}

func TestMockUpdateJobFull_NotFound(t *testing.T) {
	m := NewMockService()
	err := m.UpdateJobFull(999, "x", 1, 2, "", nil, true, nil)
	if err == nil {
		t.Error("expected error for non-existent job")
	}
}

func TestMockUpdateJobFull_ErrorInjection(t *testing.T) {
	m := NewMockService()
	m.Jobs = []Job{{ID: 1, Name: "test"}}
	m.UpdateJobFullErr = errBadCredentials{}

	err := m.UpdateJobFull(1, "x", 1, 2, "", nil, true, nil)
	if err == nil {
		t.Error("expected injected error")
	}
}

// ─── GetClearDestStatus tests ─────────────────────────────────────────────────

func TestMockGetClearDestStatus_Default(t *testing.T) {
	m := NewMockService()
	running, err := m.GetClearDestStatus(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("default should be not running")
	}
}

func TestMockGetClearDestStatus_Running(t *testing.T) {
	m := NewMockService()
	m.ClearDestRunning = true
	running, _ := m.GetClearDestStatus(1)
	if !running {
		t.Error("should report running when ClearDestRunning=true")
	}
}

func TestMockGetClearDestStatus_ErrorInjection(t *testing.T) {
	m := NewMockService()
	m.GetClearDestStatusErr = errBadCredentials{}
	_, err := m.GetClearDestStatus(1)
	if err == nil {
		t.Error("expected injected error")
	}
}

// ─── RecoverFromClearDest tests ───────────────────────────────────────────────

func TestMockRecoverFromClearDest_Success(t *testing.T) {
	m := NewMockService()
	if err := m.RecoverFromClearDest(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Calls["RecoverFromClearDest"] != 1 {
		t.Error("call not recorded")
	}
}

func TestMockRecoverFromClearDest_ErrorInjection(t *testing.T) {
	m := NewMockService()
	m.RecoverFromClearDestErr = errBadCredentials{}
	if err := m.RecoverFromClearDest(1); err == nil {
		t.Error("expected injected error")
	}
}

// ─── RunMode validation tests ─────────────────────────────────────────────────

func TestNewManager_InvalidRunMode(t *testing.T) {
	_, err := New(Config{
		DBURL:   "postgres://fake:fake@localhost/fake",
		RunMode: "invalid-mode",
	})
	if err == nil {
		t.Error("expected error for invalid run mode")
	}
	if !strings.Contains(err.Error(), "invalid run mode") {
		t.Errorf("error should mention invalid run mode, got: %v", err)
	}
}

func TestNewManager_ValidRunModes(t *testing.T) {
	for _, mode := range []string{"dev", "prod", "staging"} {
		// This will fail on DB connect but should pass the runMode check
		_, err := New(Config{
			DBURL:   "postgres://fake:fake@localhost:0/fake?connect_timeout=1",
			RunMode: mode,
		})
		if err != nil && strings.Contains(err.Error(), "invalid run mode") {
			t.Errorf("mode %q should be valid but got: %v", mode, err)
		}
	}
}
