use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

/// Render a centered confirmation dialog as an overlay.
pub fn render(frame: &mut Frame, title: &str, message: &str, yes_selected: bool) {
    let area = frame.area();

    // Dialog dimensions
    let dialog_width = 50u16.min(area.width.saturating_sub(4));
    let dialog_height = 7u16.min(area.height.saturating_sub(4));

    let x = (area.width.saturating_sub(dialog_width)) / 2;
    let y = (area.height.saturating_sub(dialog_height)) / 2;

    let dialog_area = Rect {
        x: area.x + x,
        y: area.y + y,
        width: dialog_width,
        height: dialog_height,
    };

    // Clear background behind the dialog
    frame.render_widget(Clear, dialog_area);

    let block = Block::default()
        .borders(Borders::ALL)
        .title(format!(" {} ", title))
        .title_style(
            Style::default()
                .fg(Color::Red)
                .add_modifier(Modifier::BOLD),
        )
        .border_style(Style::default().fg(Color::Red));

    frame.render_widget(block.clone(), dialog_area);
    let inner = block.inner(dialog_area);

    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(0),    // Message
            Constraint::Length(3), // Buttons
        ])
        .split(inner);

    // Message
    let msg_para = Paragraph::new(message)
        .style(Style::default().fg(Color::White))
        .wrap(Wrap { trim: true })
        .alignment(Alignment::Center);
    frame.render_widget(msg_para, chunks[0]);

    // Buttons
    let button_chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(50),
            Constraint::Percentage(50),
        ])
        .split(chunks[1]);

    let yes_style = if yes_selected {
        Style::default()
            .fg(Color::Black)
            .bg(Color::Red)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Red)
    };

    let no_style = if !yes_selected {
        Style::default()
            .fg(Color::Black)
            .bg(Color::Green)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Green)
    };

    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" Yes", yes_style)))
            .alignment(Alignment::Center)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_style(if yes_selected {
                        Style::default().fg(Color::Red)
                    } else {
                        Style::default().fg(Color::DarkGray)
                    }),
            ),
        button_chunks[0],
    );

    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" No", no_style)))
            .alignment(Alignment::Center)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_style(if !yes_selected {
                        Style::default().fg(Color::Green)
                    } else {
                        Style::default().fg(Color::DarkGray)
                    }),
            ),
        button_chunks[1],
    );
}
