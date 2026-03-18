mod confirm_dialog;
mod dashboard;
mod destination_form;
mod destinations;
mod jobs;
mod login;
mod source_form;
mod sources;

use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::{App, Screen, Tab};

/// Main render entry point — routes to login or main tabs based on auth state.
pub fn render(frame: &mut Frame, app: &App) {
    match &app.screen {
        Screen::Login => {
            let bg = Block::default().style(Style::default().bg(Color::Black));
            frame.render_widget(bg, frame.area());
            login::render(frame, app);
        }
        Screen::Main => render_main(frame, app),
        Screen::SourceForm => render_source_form(frame, app),
        Screen::DestinationForm => render_dest_form(frame, app),
        Screen::ConfirmDialog => {
            // Render main underneath, then overlay the dialog
            render_main(frame, app);
            confirm_dialog::render(
                frame,
                &app.confirm_dialog.title,
                &app.confirm_dialog.message,
                app.confirm_dialog.yes_selected,
            );
        }
    }
}

/// Render the source form screen.
fn render_source_form(frame: &mut Frame, app: &App) {
    // Draw dark background
    let bg = Block::default().style(Style::default().bg(Color::Black));
    frame.render_widget(bg, frame.area());
    source_form::render(frame, frame.area(), app);
}

/// Render the destination form screen.
fn render_dest_form(frame: &mut Frame, app: &App) {
    let bg = Block::default().style(Style::default().bg(Color::Black));
    frame.render_widget(bg, frame.area());
    destination_form::render(frame, frame.area(), app);
}

/// Render the main tabbed interface.
fn render_main(frame: &mut Frame, app: &App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3), // Tab bar
            Constraint::Min(0),    // Content
            Constraint::Length(1), // Status bar
        ])
        .split(frame.area());

    // Tab bar
    let titles: Vec<Line> = Tab::ALL
        .iter()
        .enumerate()
        .map(|(i, tab)| {
            let style = if *tab == app.active_tab {
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(Color::DarkGray)
            };
            Line::from(format!(" {} {} ", i + 1, tab.title())).style(style)
        })
        .collect();

    let tab_title = if app.auth.logged_in {
        format!(" OLake TUI  [{}] ", app.auth.username)
    } else {
        " OLake TUI ".to_string()
    };

    let tabs = Tabs::new(titles)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .title(tab_title)
                .title_style(
                    Style::default()
                        .fg(Color::Cyan)
                        .add_modifier(Modifier::BOLD),
                ),
        )
        .select(
            Tab::ALL
                .iter()
                .position(|t| *t == app.active_tab)
                .unwrap_or(0),
        )
        .highlight_style(Style::default().fg(Color::Yellow));

    frame.render_widget(tabs, chunks[0]);

    // Tab content
    match app.active_tab {
        Tab::Dashboard => dashboard::render(frame, chunks[1], app),
        Tab::Sources => sources::render(frame, chunks[1], app),
        Tab::Destinations => destinations::render(frame, chunks[1], app),
        Tab::Jobs => jobs::render(frame, chunks[1], app),
    }

    // Status bar — show error (red) or info (green) or default hint
    let status_text = if let Some(err) = &app.error_message {
        Line::from(vec![
            Span::styled(" ✗ ", Style::default().fg(Color::Red)),
            Span::styled(err.as_str(), Style::default().fg(Color::Red)),
        ])
    } else if let Some(info) = &app.info_message {
        Line::from(vec![
            Span::styled(" ✓ ", Style::default().fg(Color::Green)),
            Span::styled(info.as_str(), Style::default().fg(Color::Green)),
        ])
    } else {
        Line::from(Span::styled(
            " Tab/1-4: navigate | a: add | e: edit | d: delete | t: test | q: quit",
            Style::default().fg(Color::DarkGray),
        ))
    };

    let status = Paragraph::new(status_text);
    frame.render_widget(status, chunks[2]);
}
