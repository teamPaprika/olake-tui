use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, _app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .title(" Sources ")
        .border_style(Style::default().fg(Color::Green));

    let content = Paragraph::new(vec![
        Line::from(""),
        Line::from("  Supported sources:"),
        Line::from(Span::styled("  • PostgreSQL", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • MySQL", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • MongoDB", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • Oracle", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • MSSQL", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • DB2", Style::default().fg(Color::Green))),
        Line::from(Span::styled("  • Kafka", Style::default().fg(Color::Yellow))),
        Line::from(Span::styled("  • S3", Style::default().fg(Color::Yellow))),
        Line::from(""),
        Line::from(Span::styled(
            "  Press 'a' to add a new source (coming soon)",
            Style::default().fg(Color::DarkGray),
        )),
    ])
    .block(block);

    frame.render_widget(content, area);
}
