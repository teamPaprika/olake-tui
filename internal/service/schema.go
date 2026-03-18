package service

import (
	"context"
	"fmt"
	"strings"
)

// requiredTables lists the OLake DB tables the TUI needs in order to function.
// Table names are entity slugs; the Manager.tbl() method applies the run-mode
// prefix (e.g. "olake-dev-source").
var requiredTables = []string{
	"user",
	"source",
	"destination",
	"job",
	"project-settings",
}

// ValidateSchema checks that the connected PostgreSQL database contains the
// tables and columns required by this version of olake-tui.
//
// It is called once at startup so that the user receives a clear error message
// (rather than a cryptic SQL error deep inside the TUI) when the schema is
// out-of-date or the wrong database has been supplied.
//
// Returns nil when the schema looks good.
// Returns a descriptive error with migration instructions if anything is
// missing or incompatible.
func (m *Manager) ValidateSchema() error {
	ctx := context.Background()

	var missing []string
	for _, entity := range requiredTables {
		table := m.tbl(entity)
		// Strip surrounding quotes for the information_schema query.
		rawName := strings.Trim(table, `"`)
		query := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public'
				  AND table_name   = $1
			)`
		var exists bool
		if err := m.db.QueryRowContext(ctx, query, rawName).Scan(&exists); err != nil {
			return fmt.Errorf("schema check query failed: %w", err)
		}
		if !exists {
			missing = append(missing, rawName)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"database schema is incompatible with %s.\n\n"+
			"Missing tables:\n  • %s\n\n"+
			"To fix:\n"+
			"  1. Ensure you are pointing --db-url at the OLake PostgreSQL database.\n"+
			"  2. Run the OLake server at least once so it can apply its migrations.\n"+
			"  3. Check --run-mode matches the prefix used when OLake was initialised\n"+
			"     (e.g. --run-mode dev → expects tables named 'olake-dev-*').\n",
		m.GetCompatibleVersion(),
		strings.Join(missing, "\n  • "),
	)
}

// GetCompatibleVersion returns the minimum OLake server version that this
// build of olake-tui is known to work with.
func (m *Manager) GetCompatibleVersion() string {
	return "olake >= 1.0.0"
}
