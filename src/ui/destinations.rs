use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, _app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .title(" Destinations ")
        .border_style(Style::default().fg(Color::Magenta));

    let content = Paragraph::new(vec![
        Line::from(""),
        Line::from("  Supported destinations:"),
        Line::from(""),
        Line::from(Span::styled("  Apache Iceberg", Style::default().fg(Color::Magenta).add_modifier(Modifier::BOLD))),
        Line::from("    Catalogs: Glue, REST, Hive, JDBC"),
        Line::from("    Storage: S3, MinIO, GCS, ADLS Gen2"),
        Line::from(""),
        Line::from(Span::styled("  Parquet", Style::default().fg(Color::Magenta).add_modifier(Modifier::BOLD))),
        Line::from("    Storage: S3, GCS, Local filesystem"),
        Line::from(""),
        Line::from(Span::styled(
            "  Press 'a' to add a new destination (coming soon)",
            Style::default().fg(Color::DarkGray),
        )),
    ])
    .block(block);

    frame.render_widget(content, area);
}
