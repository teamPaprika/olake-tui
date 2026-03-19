package service

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// MigrateSchema creates all OLake tables if they don't exist, matching the
// exact schema that BFF's Beego ORM (RunSyncdb) would produce.
//
// This allows olake-tui to be used standalone without the BFF server — the
// TUI can bootstrap its own database.
//
// Safe to call repeatedly: every statement uses IF NOT EXISTS / ON CONFLICT.
func (m *Manager) MigrateSchema() error {
	ctx := context.Background()

	stmts := m.migrationStatements()
	for i, stmt := range stmts {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := m.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration step %d failed: %w\nSQL: %s", i+1, err, stmt)
		}
	}

	return nil
}

// SeedAdminUser creates a default admin user if no users exist.
// password is bcrypt-hashed before insertion, matching BFF's auth flow.
func (m *Manager) SeedAdminUser(username, password string) error {
	ctx := context.Background()
	tblUser := m.tbl("user")

	// Check if any user exists
	var count int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, tblUser)
	if err := m.db.QueryRowContext(ctx, q).Scan(&count); err != nil {
		return fmt.Errorf("check users: %w", err)
	}
	if count > 0 {
		return nil // users already exist, skip seeding
	}

	// Hash password (bcrypt cost 10, same as BFF)
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	q = fmt.Sprintf(`
		INSERT INTO %s (username, password, email, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (username) DO NOTHING`,
		tblUser)
	_, err = m.db.ExecContext(ctx, q, username, string(hashed), username+"@olake.local")
	if err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	return nil
}

// migrationStatements returns the ordered DDL needed to bootstrap the OLake schema.
// Column types, constraints, and naming match BFF's Beego ORM model definitions.
func (m *Manager) migrationStatements() []string {
	tblUser := m.tbl("user")
	tblSource := m.tbl("source")
	tblDest := m.tbl("destination")
	tblJob := m.tbl("job")
	tblSettings := m.tbl("project-settings")
	tblCatalog := m.tbl("catalog")

	return []string{
		// ── User table ────────────────────────────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id            SERIAL PRIMARY KEY,
			username      VARCHAR(100) NOT NULL UNIQUE,
			password      VARCHAR(100) NOT NULL,
			email         VARCHAR(100) NOT NULL UNIQUE,
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at    TIMESTAMP WITH TIME ZONE
		)`, tblUser),

		// ── Source table ──────────────────────────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id            SERIAL PRIMARY KEY,
			name          VARCHAR(255) NOT NULL,
			type          VARCHAR(100) NOT NULL DEFAULT '',
			version       VARCHAR(50)  NOT NULL DEFAULT '',
			config        JSONB        NOT NULL DEFAULT '{}',
			project_id    VARCHAR(100) NOT NULL DEFAULT '',
			created_by_id INTEGER      REFERENCES %s(id),
			updated_by_id INTEGER      REFERENCES %s(id),
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at    TIMESTAMP WITH TIME ZONE
		)`, tblSource, tblUser, tblUser),

		// ── Destination table ─────────────────────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id            SERIAL PRIMARY KEY,
			name          VARCHAR(255) NOT NULL,
			dest_type     VARCHAR(100) NOT NULL DEFAULT '',
			version       VARCHAR(50)  NOT NULL DEFAULT '',
			config        JSONB        NOT NULL DEFAULT '{}',
			project_id    VARCHAR(100) NOT NULL DEFAULT '',
			created_by_id INTEGER      REFERENCES %s(id),
			updated_by_id INTEGER      REFERENCES %s(id),
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at    TIMESTAMP WITH TIME ZONE
		)`, tblDest, tblUser, tblUser),

		// ── Job table ─────────────────────────────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id                SERIAL PRIMARY KEY,
			name              VARCHAR(100) NOT NULL,
			source_id         INTEGER      REFERENCES %s(id),
			dest_id           INTEGER      REFERENCES %s(id),
			active            BOOLEAN      NOT NULL DEFAULT true,
			frequency         VARCHAR(100) NOT NULL DEFAULT '',
			streams_config    JSONB        NOT NULL DEFAULT '[]',
			state             JSONB        NOT NULL DEFAULT '{}',
			advanced_settings JSONB,
			project_id        VARCHAR(100) NOT NULL DEFAULT '',
			created_by_id     INTEGER      REFERENCES %s(id),
			updated_by_id     INTEGER      REFERENCES %s(id),
			created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at        TIMESTAMP WITH TIME ZONE
		)`, tblJob, tblSource, tblDest, tblUser, tblUser),

		// ── Project settings table ────────────────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id              SERIAL PRIMARY KEY,
			project_id      VARCHAR(100) NOT NULL UNIQUE,
			webhook_alert_url VARCHAR(512) DEFAULT '',
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at      TIMESTAMP WITH TIME ZONE
		)`, tblSettings),

		// ── Catalog table (connector specs) ───────────────────────────────
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id         SERIAL PRIMARY KEY,
			type       VARCHAR(50)  NOT NULL DEFAULT '',
			name       VARCHAR(100) NOT NULL DEFAULT '',
			specs      JSONB        NOT NULL DEFAULT '{}',
			version    VARCHAR(50)  NOT NULL DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			deleted_at TIMESTAMP WITH TIME ZONE
		)`, tblCatalog),

		// ── Session table (for BFF compatibility) ─────────────────────────
		`CREATE TABLE IF NOT EXISTS session (
			session_key    VARCHAR(64) PRIMARY KEY,
			session_data   BYTEA,
			session_expiry TIMESTAMP WITH TIME ZONE
		)`,
	}
}
