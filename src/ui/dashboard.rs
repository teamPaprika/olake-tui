use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, _app: &App) {
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

    // Quick stats placeholder
    let stats = Table::new(
        vec![
            Row::new(vec!["Sources", "0"]),
            Row::new(vec!["Destinations", "0"]),
            Row::new(vec!["Jobs", "0"]),
            Row::new(vec!["Running", "0"]),
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
