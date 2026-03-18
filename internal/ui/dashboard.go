package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/datazip-inc/olake-tui/internal/service"
)

// DashboardStats holds summary counts for the dashboard widget.
type DashboardStats struct {
	Sources      int
	Destinations int
	Jobs         int
	ActiveJobs   int
	RunningJobs  int
}

// ComputeDashboardStats derives stats from the current data.
func ComputeDashboardStats(sources []service.Source, dests []service.Destination, jobs []service.Job) DashboardStats {
	running := 0
	active := 0
	for _, j := range jobs {
		if j.Activate {
			active++
		}
		if j.LastRunState == "WORKFLOW_EXECUTION_STATUS_RUNNING" {
			running++
		}
	}
	return DashboardStats{
		Sources:      len(sources),
		Destinations: len(dests),
		Jobs:         len(jobs),
		ActiveJobs:   active,
		RunningJobs:  running,
	}
}

// RenderDashboard renders a compact overview bar.
func RenderDashboard(stats DashboardStats, username string) string {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 2).
		Width(18)

	activeCardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Padding(0, 2).
		Width(18)

	card := func(label, value string, highlighted bool) string {
		style := cardStyle
		if highlighted {
			style = activeCardStyle
		}
		v := StyleTitle.Render(value)
		l := StyleMuted.Render(label)
		return style.Render(lipgloss.JoinVertical(lipgloss.Left, v, l))
	}

	cards := lipgloss.JoinHorizontal(lipgloss.Top,
		card("Sources", fmt.Sprintf("%d", stats.Sources), false),
		"  ",
		card("Destinations", fmt.Sprintf("%d", stats.Destinations), false),
		"  ",
		card("Jobs", fmt.Sprintf("%d", stats.Jobs), false),
		"  ",
		card("Active Jobs", fmt.Sprintf("%d", stats.ActiveJobs), stats.ActiveJobs > 0),
		"  ",
		card("Running", fmt.Sprintf("%d", stats.RunningJobs), stats.RunningJobs > 0),
	)

	greeting := lipgloss.JoinHorizontal(lipgloss.Center,
		StyleLogo.Render("⬡ OLake"),
		"  ",
		StyleMuted.Render(fmt.Sprintf("logged in as %s", username)),
	)

	return lipgloss.JoinVertical(lipgloss.Left, greeting, "", cards)
}
