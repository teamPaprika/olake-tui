package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Simulated screen renders for screenshot purposes
// These mirror the actual UI code output

func main() {
	width := 80
	
	fmt.Println("=== SCREEN 1: LOGIN ===")
	fmt.Println(renderLogin(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 2: DASHBOARD ===")
	fmt.Println(renderDashboard(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 3: SOURCES LIST ===")
	fmt.Println(renderSources(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 4: DESTINATIONS LIST ===")
	fmt.Println(renderDestinations(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 5: JOBS LIST ===")
	fmt.Println(renderJobs(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 6: JOB WIZARD (Step 1) ===")
	fmt.Println(renderJobWizardStep1(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 7: JOB WIZARD (Step 4 - Streams) ===")
	fmt.Println(renderJobWizardStreams(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 8: JOB LOGS ===")
	fmt.Println(renderJobLogs(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 9: SOURCE FORM ===")
	fmt.Println(renderSourceForm(width))
	fmt.Println()
	
	fmt.Println("=== SCREEN 10: JOB SETTINGS ===")
	fmt.Println(renderJobSettings(width))
}

var (
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	gray    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	blue    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	bold    = lipgloss.NewStyle().Bold(true)
	
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2)
	
	activeTab = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		Underline(true)
	
	inactiveTab = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	
	selectedRow = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Bold(true)
)

func tabBar(active int) string {
	tabs := []string{"1 Dashboard", "2 Sources", "3 Destinations", "4 Jobs", "5 Settings"}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		if i == active {
			parts[i] = activeTab.Render(t)
		} else {
			parts[i] = inactiveTab.Render(t)
		}
	}
	header := cyan.Bold(true).Render(" ⚡ OLake TUI ") + "  " + strings.Join(parts, "  │  ")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(78).
		Render(header)
}

func statusBar(text string) string {
	return gray.Render(text)
}

func renderLogin(w int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(40).
		Padding(1, 3).
		Align(lipgloss.Center)

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		cyan.Bold(true).Render("⚡ OLake TUI"),
		yellow.Render("Username:"),
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("3")).Width(30).Render("  admin█"),
		gray.Render("Password:"),
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Width(30).Render("  ••••••••"),
		gray.Render("[ Enter: login | Tab: switch field | Esc: quit ]"),
	)
	return box.Render(content)
}

func renderDashboard(w int) string {
	var b strings.Builder
	b.WriteString(tabBar(0) + "\n\n")
	
	welcome := boxStyle.Width(76).Render(
		cyan.Bold(true).Render("  ⚡ OLake TUI v0.2.0-direct") + "\n\n" +
		"  Terminal UI for OLake — Fastest open-source data replication\n" +
		"  Database → Apache Iceberg / Parquet\n\n" +
		gray.Render("  Connected to PostgreSQL ✓  |  Temporal ✓"))
	b.WriteString(welcome + "\n\n")
	
	stats := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(76).
		Padding(0, 2).
		Render(
		bold.Render("  Overview") + "\n" +
		"  ─────────────────────────────\n" +
		fmt.Sprintf("  Sources          %s\n", green.Render("3")) +
		fmt.Sprintf("  Destinations     %s\n", magenta.Render("2")) +
		fmt.Sprintf("  Jobs             %s\n", blue.Render("5")) +
		fmt.Sprintf("  Running          %s", green.Render("1")))
	b.WriteString(stats + "\n\n")
	b.WriteString(statusBar(" Tab/1-5: navigate | q: quit"))
	return b.String()
}

func renderSources(w int) string {
	var b strings.Builder
	b.WriteString(tabBar(1) + "\n\n")
	
	header := fmt.Sprintf("  %-20s %-14s %-10s %-20s", "NAME", "TYPE", "STATUS", "CREATED")
	b.WriteString(bold.Render(header) + "\n")
	b.WriteString(gray.Render("  " + strings.Repeat("─", 70)) + "\n")
	b.WriteString(selectedRow.Render(fmt.Sprintf("  %-20s %-14s %-10s %-20s", "prod-postgres", "PostgreSQL", green.Render("●  Active"), "2024-03-15 09:30")) + "\n")
	b.WriteString(fmt.Sprintf("  %-20s %-14s %-10s %-20s\n", "staging-mysql", "MySQL", green.Render("●  Active"), "2024-03-14 14:22"))
	b.WriteString(fmt.Sprintf("  %-20s %-14s %-10s %-20s\n", "analytics-mongo", "MongoDB", yellow.Render("●  Warn"), "2024-03-10 11:00"))
	
	b.WriteString("\n\n")
	b.WriteString(statusBar(" a:add | e:edit | d:delete | t:test | r:refresh | Tab:next"))
	return b.String()
}

func renderDestinations(w int) string {
	var b strings.Builder
	b.WriteString(tabBar(2) + "\n\n")
	
	header := fmt.Sprintf("  %-20s %-14s %-10s %-20s", "NAME", "TYPE", "STATUS", "CREATED")
	b.WriteString(bold.Render(header) + "\n")
	b.WriteString(gray.Render("  " + strings.Repeat("─", 70)) + "\n")
	b.WriteString(selectedRow.Render(fmt.Sprintf("  %-20s %-14s %-10s %-20s", "lakehouse-iceberg", "Iceberg", green.Render("●  Active"), "2024-03-15 10:00")) + "\n")
	b.WriteString(fmt.Sprintf("  %-20s %-14s %-10s %-20s\n", "archive-parquet", "Parquet", green.Render("●  Active"), "2024-03-12 08:15"))
	
	b.WriteString("\n\n")
	b.WriteString(statusBar(" a:add | e:edit | d:delete | t:test | r:refresh | Tab:next"))
	return b.String()
}

func renderJobs(w int) string {
	var b strings.Builder
	b.WriteString(tabBar(3) + "\n\n")
	
	header := fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s", "NAME", "SOURCE → DEST", "STATUS", "SCHEDULE", "LAST SYNC")
	b.WriteString(bold.Render(header) + "\n")
	b.WriteString(gray.Render("  " + strings.Repeat("─", 76)) + "\n")
	b.WriteString(selectedRow.Render(fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s", "pg-to-iceberg", "prod-pg → lakehouse", green.Render("● Running"), "Every 30m", "2 min ago")) + "\n")
	b.WriteString(fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s\n", "mysql-sync", "staging → lakehouse", blue.Render("● Done"), "Every 1h", "28 min ago"))
	b.WriteString(fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s\n", "mongo-archive", "analytics → parquet", red.Render("● Failed"), "Daily 02:00", "18h ago"))
	b.WriteString(fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s\n", "kafka-ingest", "events → lakehouse", gray.Render("● Inactive"), "Manual", "3d ago"))
	b.WriteString(fmt.Sprintf("  %-16s %-22s %-10s %-12s %-14s\n", "s3-import", "s3-bucket → parquet", blue.Render("● Done"), "Weekly Mon", "5d ago"))
	
	b.WriteString("\n\n")
	b.WriteString(statusBar(" n:new | Enter:detail | S:settings | s:sync | c:cancel | l:logs | d:delete | r:refresh"))
	return b.String()
}

func renderJobWizardStep1(w int) string {
	var b strings.Builder
	
	// Step indicator
	steps := fmt.Sprintf("  %s  →  %s  →  %s  →  %s",
		yellow.Bold(true).Render("① Config"),
		gray.Render("② Source"),
		gray.Render("③ Destination"),
		gray.Render("④ Streams"))
	
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(78).
		Render(cyan.Bold(true).Render(" ⚡ New Job") + "  " + steps) + "\n\n")
	
	b.WriteString(yellow.Render("  Job Name:") + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("3")).Width(50).Render("  pg-to-iceberg-daily█") + "\n\n")
	
	b.WriteString(bold.Render("  Select Source:") + "\n")
	b.WriteString(selectedRow.Render("  → prod-postgres (PostgreSQL)") + "\n")
	b.WriteString("    staging-mysql (MySQL)\n")
	b.WriteString("    analytics-mongo (MongoDB)\n\n")
	
	b.WriteString(bold.Render("  Select Destination:") + "\n")
	b.WriteString("    lakehouse-iceberg (Iceberg)\n")
	b.WriteString("    archive-parquet (Parquet)\n\n")
	
	b.WriteString(statusBar(" Tab:next step | j/k:navigate | Enter:select | Esc:cancel"))
	return b.String()
}

func renderJobWizardStreams(w int) string {
	var b strings.Builder
	
	steps := fmt.Sprintf("  %s  →  %s  →  %s  →  %s",
		green.Render("✓ Config"),
		green.Render("✓ Source"),
		green.Render("✓ Destination"),
		yellow.Bold(true).Render("④ Streams"))
	
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(78).
		Render(cyan.Bold(true).Render(" ⚡ New Job") + "  " + steps) + "\n\n")
	
	b.WriteString(bold.Render("  Select Streams") + "  " + gray.Render("(8/12 selected)") + "\n")
	b.WriteString(gray.Render("  " + strings.Repeat("─", 70)) + "\n")
	
	b.WriteString(selectedRow.Render("  [x] public.users                    CDC          cursor: updated_at") + "\n")
	b.WriteString("  [x] public.orders                   CDC          cursor: id\n")
	b.WriteString("  [x] public.products                 Full Refresh\n")
	b.WriteString("  [x] public.payments                 Incremental  cursor: created_at\n")
	b.WriteString("  [ ] public.sessions                 —\n")
	b.WriteString("  [x] public.invoices                 CDC          cursor: updated_at\n")
	b.WriteString("  [ ] public.temp_data                —\n")
	b.WriteString("  [x] public.customers                CDC          cursor: modified\n")
	b.WriteString("  [ ] analytics.events                —\n")
	b.WriteString("  [x] analytics.pageviews             Full Refresh\n")
	b.WriteString("  [ ] analytics.raw_logs              —\n")
	b.WriteString("  [x] analytics.conversions           Incremental  cursor: timestamp\n")
	
	b.WriteString("\n")
	b.WriteString(statusBar(" Space:toggle | a:all | Enter:config stream | /:search | Ctrl+Enter:create job | Esc:back"))
	return b.String()
}

func renderJobLogs(w int) string {
	var b strings.Builder
	
	header := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(78).
		Render(
		cyan.Bold(true).Render(" Job: pg-to-iceberg") + "  │  " +
		gray.Render("Task: abc123") + "  │  " +
		green.Render("● Running"))
	b.WriteString(header + "\n")
	
	// Filter bar
	b.WriteString(fmt.Sprintf("  Level: %s %s %s %s  │  Search: %s\n",
		yellow.Bold(true).Render("[ALL]"),
		gray.Render(" INFO "),
		gray.Render(" WARN "),
		gray.Render(" ERROR"),
		gray.Render("_______________")))
	b.WriteString(gray.Render("  " + strings.Repeat("─", 76)) + "\n")
	
	// Log entries
	logs := []struct{ ts, level, msg string }{
		{"22:30:01", "INFO", "Starting sync for job pg-to-iceberg..."},
		{"22:30:01", "INFO", "Connecting to source: prod-postgres (PostgreSQL)"},
		{"22:30:02", "INFO", "Connection established successfully"},
		{"22:30:02", "INFO", "Discovering schema changes..."},
		{"22:30:03", "INFO", "Found 8 streams to sync"},
		{"22:30:03", "INFO", "Starting CDC replication for public.users"},
		{"22:30:04", "INFO", "Processing batch: 1,024 records"},
		{"22:30:05", "WARN", "Slow query detected on public.orders (>2s)"},
		{"22:30:06", "INFO", "Written 1,024 records to Iceberg (public.users)"},
		{"22:30:06", "INFO", "Starting CDC replication for public.orders"},
		{"22:30:07", "INFO", "Processing batch: 2,048 records"},
		{"22:30:08", "INFO", "Written 2,048 records to Iceberg (public.orders)"},
		{"22:30:09", "ERROR", "Connection timeout on public.payments — retrying (1/3)"},
		{"22:30:12", "INFO", "Reconnected to public.payments"},
		{"22:30:13", "INFO", "Processing batch: 512 records"},
	}
	
	for _, l := range logs {
		var levelStr string
		switch l.level {
		case "INFO":
			levelStr = gray.Render("INFO ")
		case "WARN":
			levelStr = yellow.Render("WARN ")
		case "ERROR":
			levelStr = red.Render("ERROR")
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n", gray.Render(l.ts), levelStr, l.msg))
	}
	
	b.WriteString("\n")
	b.WriteString(statusBar(" j/k:scroll | ←→:level filter | /:search | d:download | p/n:page | q:close"))
	return b.String()
}

func renderSourceForm(w int) string {
	var b strings.Builder
	
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Width(78).
		Render(green.Bold(true).Render(" New Source")) + "\n\n")
	
	b.WriteString(bold.Render("  Source Type: ") + yellow.Render("◀ ") + green.Bold(true).Render("PostgreSQL") + yellow.Render(" ▶") + "\n\n")
	
	b.WriteString(yellow.Render("  Name:") + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("3")).Width(50).Render("  prod-postgres█") + "\n\n")
	
	fields := []struct{ label, value string; masked bool }{
		{"Host", "db.example.com", false},
		{"Port", "5432", false},
		{"Username", "olake_user", false},
		{"Password", "••••••••••", true},
		{"Database", "production", false},
	}
	
	for _, f := range fields {
		b.WriteString(gray.Render(fmt.Sprintf("  %s:", f.label)) + "\n")
		b.WriteString("  " + lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Width(50).Render("  " + f.value) + "\n")
	}
	
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s    %s    %s\n",
		lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Padding(0, 2).Render(" Test Connection "),
		lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0")).Padding(0, 2).Render(" Save "),
		gray.Render("Esc: cancel")))
	
	return b.String()
}

func renderJobSettings(w int) string {
	var b strings.Builder
	
	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Width(78).
		Render(blue.Bold(true).Render(" Job Settings: pg-to-iceberg")) + "\n\n")
	
	b.WriteString(bold.Render("  Job Name:") + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Width(50).Render("  pg-to-iceberg") + "\n\n")
	
	b.WriteString(bold.Render("  Schedule") + "\n")
	b.WriteString(fmt.Sprintf("  Frequency: %s %s %s %s %s %s\n",
		gray.Render("Manual"),
		gray.Render("│"),
		yellow.Bold(true).Render("Every 30m"),
		gray.Render("│"),
		gray.Render("Daily"),
		gray.Render("│ Weekly │ Custom")))
	b.WriteString(gray.Render("  Next run: in 28 minutes") + "\n\n")
	
	b.WriteString(bold.Render("  Status: ") + green.Render("● Active") + "  " + gray.Render("[p to pause]") + "\n\n")
	
	b.WriteString(gray.Render("  ────────────────────────────────────────") + "\n")
	b.WriteString(fmt.Sprintf("  %s    %s\n",
		red.Render("[ Clear Destination ]"),
		red.Render("[ Delete Job ]")))
	
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s    %s\n",
		lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("0")).Padding(0, 2).Render(" Save "),
		gray.Render("Esc: cancel")))
	
	return b.String()
}
