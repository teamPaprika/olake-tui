use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, app: &App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(8),
            Constraint::Min(0),
        ])
        .split(area);

    // Welcome / summary
    let welcome = Paragraph::new(vec![
        Line::from(""),
        Line::from(Span::styled(
            "  ⚡ OLake TUI",
            Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
        )),
        Line::from(""),
        Line::from("  Fastest open-source data replication tool"),
        Line::from("  Database → Apache Iceberg / Parquet"),
        Line::from(""),
        Line::from(Span::styled(
            "  Connect a source to get started.",
            Style::default().fg(Color::DarkGray),
        )),
    ])
    .block(
        Block::default()
            .borders(Borders::ALL)
            .title(" Dashboard ")
            .border_style(Style::default().fg(Color::Cyan)),
    );
    frame.render_widget(welcome, chunks[0]);

    // Count running jobs
    let running_jobs = app
        .jobs
        .iter()
        .filter(|j| j.last_run_state.to_lowercase() == "running")
        .count();

    // Pre-allocate strings so they live long enough
    let sources_count = app.sources.len().to_string();
    let destinations_count = app.destinations.len().to_string();
    let jobs_count = app.jobs.len().to_string();
    let running_count = running_jobs.to_string();

    // Quick stats with real data
    let stats = Table::new(
        vec![
            Row::new(vec!["Sources", sources_count.as_str()]),
            Row::new(vec!["Destinations", destinations_count.as_str()]),
            Row::new(vec!["Jobs", jobs_count.as_str()]),
            Row::new(vec!["Running", running_count.as_str()]),
        ],
        [Constraint::Length(20), Constraint::Length(10)],
    )
    .header(
        Row::new(vec!["Metric", "Count"])
            .style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
    )
    .block(
        Block::default()
            .borders(Borders::ALL)
            .title(" Overview "),
    );
    frame.render_widget(stats, chunks[1]);
}
