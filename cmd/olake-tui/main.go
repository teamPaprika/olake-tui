// Command olake-tui is the terminal user interface for OLake data pipelines.
//
// It connects directly to the OLake PostgreSQL database and Temporal cluster,
// bypassing the BFF HTTP layer entirely.
//
// Usage:
//
//	olake-tui [flags]
//
// Flags:
//
//	--db-url          PostgreSQL connection string (overrides OLAKE_DB_URL)
//	--temporal-host   Temporal frontend address (overrides TEMPORAL_ADDRESS, default: localhost:7233)
//	--project-id      OLake project ID (default: 123)
//	--run-mode        Beego run mode for table names: dev|prod|staging (default: dev)
//	--encryption-key  AES encryption key (overrides OLAKE_SECRET_KEY)
//	--version         Print version and exit
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/app"
	"github.com/datazip-inc/olake-tui/internal/service"
)

const version = "0.2.0-direct"

func main() {
	var (
		dbURL         = flag.String("db-url", envOr("OLAKE_DB_URL", ""), "PostgreSQL connection string")
		temporalHost  = flag.String("temporal-host", envOr("TEMPORAL_ADDRESS", service.DefaultTemporalHost), "Temporal frontend address")
		projectID     = flag.String("project-id", envOr("OLAKE_PROJECT_ID", service.DefaultProjectID), "OLake project ID")
		runMode       = flag.String("run-mode", envOr("OLAKE_RUN_MODE", service.DefaultRunMode), "Beego run mode (dev|prod|staging)")
		encryptionKey = flag.String("encryption-key", envOr("OLAKE_SECRET_KEY", ""), "AES encryption key")
		showVersion   = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("olake-tui v%s\n", version)
		os.Exit(0)
	}

	if *dbURL == "" {
		fmt.Fprintln(os.Stderr, "Error: --db-url (or OLAKE_DB_URL env var) is required")
		fmt.Fprintln(os.Stderr, "Example: --db-url 'postgres://user:pass@localhost:5432/olake?sslmode=disable'")
		os.Exit(1)
	}

	// Create service manager (direct DB + Temporal connection)
	svc, err := service.New(service.Config{
		DBURL:         *dbURL,
		TemporalHost:  *temporalHost,
		ProjectID:     *projectID,
		RunMode:       *runMode,
		EncryptionKey: *encryptionKey,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer svc.Close()

	// Create root model
	model := app.New(svc)

	// Run Bubble Tea program in alternate screen (fullscreen TUI)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// envOr returns the environment variable value or a fallback default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
