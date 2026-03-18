use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::{App, LoginField};

/// Render the login screen.
pub fn render(frame: &mut Frame, app: &App) {
    let area = frame.area();

    // Center the login box
    let outer = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Percentage(25),
            Constraint::Length(18),
            Constraint::Min(0),
        ])
        .split(area);

    let horizontal = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(30),
            Constraint::Percentage(40),
            Constraint::Percentage(30),
        ])
        .split(outer[1]);

    let login_area = horizontal[1];

    // Clear background in the login box area
    frame.render_widget(Clear, login_area);

    // Outer border / title
    let block = Block::default()
        .borders(Borders::ALL)
        .title(" OLake TUI — Login ")
        .title_style(
            Style::default()
                .fg(Color::Cyan)
                .add_modifier(Modifier::BOLD),
        )
        .border_style(Style::default().fg(Color::Cyan));

    frame.render_widget(block, login_area);

    // Inner layout: logo + fields + hint
    let inner = Layout::default()
        .direction(Direction::Vertical)
        .margin(2)
        .constraints([
            Constraint::Length(2), // logo/tagline
            Constraint::Length(1), // spacer
            Constraint::Length(3), // username field
            Constraint::Length(1), // spacer
            Constraint::Length(3), // password field
            Constraint::Length(1), // spacer
            Constraint::Length(1), // error / spinner
            Constraint::Min(0),    // hint
        ])
        .split(login_area);

    // Logo
    let logo = Paragraph::new(Span::styled(
        "⚡ OLake TUI",
        Style::default()
            .fg(Color::Cyan)
            .add_modifier(Modifier::BOLD),
    ))
    .alignment(Alignment::Center);
    frame.render_widget(logo, inner[0]);

    // Username field
    let username_style = if app.login_form.focused_field == LoginField::Username {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::White)
    };
    let username_border = if app.login_form.focused_field == LoginField::Username {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    let username_widget = Paragraph::new(app.login_form.username.as_str())
        .style(username_style)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .title(" Username ")
                .border_style(username_border),
        );
    frame.render_widget(username_widget, inner[2]);

    // Password field (mask characters)
    let password_display: String = "•".repeat(app.login_form.password.len());
    let password_style = if app.login_form.focused_field == LoginField::Password {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::White)
    };
    let password_border = if app.login_form.focused_field == LoginField::Password {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    let password_widget = Paragraph::new(password_display.as_str())
        .style(password_style)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .title(" Password ")
                .border_style(password_border),
        );
    frame.render_widget(password_widget, inner[4]);

    // Error / loading spinner
    let status_line = if app.login_form.loading.loading {
        Paragraph::new(Line::from(vec![
            Span::styled(
                app.login_form.loading.spinner(),
                Style::default().fg(Color::Cyan),
            ),
            Span::raw(" Logging in…"),
        ]))
        .alignment(Alignment::Center)
    } else if let Some(err) = &app.login_form.error {
        Paragraph::new(Span::styled(
            err.as_str(),
            Style::default().fg(Color::Red),
        ))
        .alignment(Alignment::Center)
    } else {
        Paragraph::new("")
    };
    frame.render_widget(status_line, inner[6]);

    // Hint
    let hint = Paragraph::new("Tab: switch field  Enter: login  Esc: quit")
        .style(Style::default().fg(Color::DarkGray))
        .alignment(Alignment::Center);
    frame.render_widget(hint, inner[7]);
}
