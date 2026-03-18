use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, _app: &App) {
    let block = Block::default()
        .borders(Borders::ALL)
        .title(" Jobs ")
        .border_style(Style::default().fg(Color::Blue));

    let content = Paragraph::new(vec![
        Line::from(""),
        Line::from("  No jobs configured yet."),
        Line::from(""),
        Line::from("  Jobs let you:"),
        Line::from("    • Connect a source → destination"),
        Line::from("    • Select streams (tables) to sync"),
        Line::from("    • Choose sync mode: Full Refresh / CDC / Incremental"),
        Line::from("    • Schedule recurring syncs"),
        Line::from("    • Monitor progress and logs"),
        Line::from(""),
        Line::from(Span::styled(
            "  Press 'n' to create a new job (coming soon)",
            Style::default().fg(Color::DarkGray),
        )),
    ])
    .block(block);

    frame.render_widget(content, area);
}
