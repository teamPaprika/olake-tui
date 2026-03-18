// Command olake-tui is the terminal user interface for OLake data pipelines.
//
// It connects to the OLake BFF server (github.com/datazip-inc/olake-ui/server)
// via HTTP and provides a Bubble Tea TUI for managing sources, destinations, and jobs.
//
// Usage:
//
//	olake-tui [--api-url http://localhost:8000]
//
// Flags:
//
//	--api-url   URL of the OLake BFF server (default: http://localhost:8000)
//	--help      Show help
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/datazip-inc/olake-tui/internal/app"
	"github.com/datazip-inc/olake-tui/internal/service"
)

const version = "0.1.0-go"

func main() {
	var (
		apiURL      = flag.String("api-url", service.DefaultAPIURL, "OLake BFF server URL")
		showVersion = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("olake-tui v%s\n", version)
		os.Exit(0)
	}

	// Create service manager (HTTP client to BFF)
	svc := service.New(*apiURL)

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
