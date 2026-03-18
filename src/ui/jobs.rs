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
    let title = if app.loading_jobs.loading {
        format!(" Jobs  {} Loading… ", app.loading_jobs.spinner())
    } else {
        format!(" Jobs ({}) ", app.jobs.len())
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .border_style(Style::default().fg(Color::Blue));

    if app.loading_jobs.loading && app.jobs.is_empty() {
        // Show spinner-only placeholder
        let spinner_text = Paragraph::new(Line::from(vec![
            Span::styled(
                format!("  {} Loading jobs…", app.loading_jobs.spinner()),
                Style::default().fg(Color::DarkGray),
            ),
        ]))
        .block(block);
        frame.render_widget(spinner_text, chunks[0]);
    } else if app.jobs.is_empty() {
        // Empty state
        let empty = Paragraph::new(vec![
            Line::from(""),
            Line::from(Span::styled(
                "  No jobs configured. Press 'n' to create a new job.",
                Style::default().fg(Color::DarkGray),
            )),
        ])
        .block(block);
        frame.render_widget(empty, chunks[0]);
    } else {
        // Table with data
        let header = Row::new(vec!["Name", "Source → Dest", "Status", "Schedule", "Last Sync"])
            .style(
                Style::default()
                    .fg(Color::Yellow)
                    .add_modifier(Modifier::BOLD),
            )
            .bottom_margin(1);

        let rows: Vec<Row> = app
            .jobs
            .iter()
            .enumerate()
            .map(|(i, job)| {
                let is_selected = i == app.selected_job_idx;

                // Color-code status
                let status = job.last_run_state.as_str();
                let status_style = match status.to_lowercase().as_str() {
                    "running" => Style::default().fg(Color::Green),
                    "failed" => Style::default().fg(Color::Red),
                    "completed" | "success" | "succeeded" => Style::default().fg(Color::Blue),
                    _ => Style::default().fg(Color::DarkGray),
                };

                let row_style = if is_selected {
                    Style::default()
                        .fg(Color::Black)
                        .bg(Color::Blue)
                        .add_modifier(Modifier::BOLD)
                } else {
                    Style::default().fg(Color::White)
                };

                let source_dest = format!(
                    "{} → {}",
                    job.source.name,
                    job.destination.name
                );

                // Truncate last_run_time to something readable
                let last_sync = if job.last_run_time.len() >= 16 {
                    job.last_run_time[..16].to_string()
                } else if job.last_run_time.is_empty() {
                    "Never".to_string()
                } else {
                    job.last_run_time.clone()
                };

                let active_indicator = if job.activate { "" } else { "⏸ " };

                let status_display = if is_selected {
                    // When selected, override status style with inverted
                    Cell::from(format!("{}{}", active_indicator, status))
                } else {
                    Cell::from(format!("{}{}", active_indicator, status))
                        .style(status_style)
                };

                Row::new(vec![
                    Cell::from(job.name.clone()),
                    Cell::from(source_dest),
                    status_display,
                    Cell::from(job.frequency.clone()),
                    Cell::from(last_sync),
                ])
                .style(row_style)
            })
            .collect();

        let widths = [
            Constraint::Percentage(20),
            Constraint::Percentage(30),
            Constraint::Percentage(15),
            Constraint::Percentage(15),
            Constraint::Percentage(20),
        ];

        let table = Table::new(rows, widths)
            .header(header)
            .block(block)
            .row_highlight_style(
                Style::default()
                    .fg(Color::Black)
                    .bg(Color::Blue)
                    .add_modifier(Modifier::BOLD),
            )
            .highlight_symbol("▶ ");

        frame.render_widget(table, chunks[0]);
    }

    // -----------------------------------------------------------------------
    // Keybindings bar
    // -----------------------------------------------------------------------
    let keybindings = Paragraph::new(Line::from(Span::styled(
        " n: new job | e: edit | s: sync now | c: cancel | d: delete | r: refresh | j/k: navigate",
        Style::default().fg(Color::DarkGray),
    )));
    frame.render_widget(keybindings, chunks[1]);
}
