use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::App;

pub fn render(frame: &mut Frame, area: Rect, app: &App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(0),    // Table
            Constraint::Length(1), // Keybindings
        ])
        .split(area);

    // -----------------------------------------------------------------------
    // Table block
    // -----------------------------------------------------------------------
    let title = if app.loading_sources.loading {
        format!(" Sources  {} Loading… ", app.loading_sources.spinner())
    } else {
        format!(" Sources ({}) ", app.sources.len())
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .border_style(Style::default().fg(Color::Green));

    if app.loading_sources.loading && app.sources.is_empty() {
        // Show spinner-only placeholder
        let spinner_text = Paragraph::new(Line::from(vec![
            Span::styled(
                format!("  {} Loading sources…", app.loading_sources.spinner()),
                Style::default().fg(Color::DarkGray),
            ),
        ]))
        .block(block);
        frame.render_widget(spinner_text, chunks[0]);
    } else if app.sources.is_empty() {
        // Empty state
        let empty = Paragraph::new(vec![
            Line::from(""),
            Line::from(Span::styled(
                "  No sources configured. Press 'a' to add one.",
                Style::default().fg(Color::DarkGray),
            )),
        ])
        .block(block);
        frame.render_widget(empty, chunks[0]);
    } else {
        // Table with data
        let header = Row::new(vec!["Name", "Type", "Status", "Created"])
            .style(
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD),
            )
            .bottom_margin(1);

        let rows: Vec<Row> = app
            .sources
            .iter()
            .enumerate()
            .map(|(i, entity)| {
                let style = if i == app.selected_source_idx {
                    Style::default()
                        .fg(Color::Black)
                        .bg(Color::Green)
                        .add_modifier(Modifier::BOLD)
                } else {
                    Style::default().fg(Color::White)
                };

                // Derive a "status" from jobs
                let status = if entity.jobs.is_empty() {
                    "No jobs"
                } else {
                    let running = entity.jobs.iter().any(|j| j.last_run_state == "running");
                    let failed = entity.jobs.iter().any(|j| j.last_run_state == "failed");
                    if running {
                        "Running"
                    } else if failed {
                        "Failed"
                    } else {
                        "Active"
                    }
                };

                // Truncate created_at to date part
                let created = if entity.created_at.len() >= 10 {
                    &entity.created_at[..10]
                } else {
                    &entity.created_at
                };

                Row::new(vec![
                    entity.name.clone(),
                    entity.entity_type.clone(),
                    status.to_string(),
                    created.to_string(),
                ])
                .style(style)
            })
            .collect();

        let widths = [
            Constraint::Percentage(35),
            Constraint::Percentage(25),
            Constraint::Percentage(20),
            Constraint::Percentage(20),
        ];

        let table = Table::new(rows, widths)
            .header(header)
            .block(block)
            .row_highlight_style(
                Style::default()
                    .fg(Color::Black)
                    .bg(Color::Green)
                    .add_modifier(Modifier::BOLD),
            )
            .highlight_symbol("▶ ");

        frame.render_widget(table, chunks[0]);
    }

    // -----------------------------------------------------------------------
    // Keybindings bar
    // -----------------------------------------------------------------------
    let keybindings = Paragraph::new(Line::from(Span::styled(
        " a: add | e: edit | d: delete | t: test | r: refresh | j/k: navigate",
        Style::default().fg(Color::DarkGray),
    )));
    frame.render_widget(keybindings, chunks[1]);
}
