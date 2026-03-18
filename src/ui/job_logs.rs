use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::{App, LogLevelFilter};

/// Spinner frames for the loading indicator.
const SPINNER_FRAMES: &[&str] = &["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

/// Color for a log level string.
fn level_style(level: &str) -> Style {
    match level.to_lowercase().as_str() {
        "warn" | "warning" => Style::default().fg(Color::Yellow),
        "error" | "fatal" => Style::default()
            .fg(Color::Red)
            .add_modifier(Modifier::BOLD),
        "debug" => Style::default().fg(Color::DarkGray),
        _ => Style::default().fg(Color::Gray), // info / unknown
    }
}

/// Truncate a string to `max` chars, appending "…" if it was cut.
fn truncate(s: &str, max: usize) -> String {
    if s.len() <= max {
        s.to_string()
    } else {
        format!("{}…", &s[..max.saturating_sub(1)])
    }
}

pub fn render(frame: &mut Frame, app: &App) {
    let area = frame.area();

    // Dark background
    let bg = Block::default().style(Style::default().bg(Color::Black));
    frame.render_widget(bg, area);

    // Layout: header | filter bar | log list | footer
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3), // header
            Constraint::Length(3), // filter bar
            Constraint::Min(0),    // log list
            Constraint::Length(1), // footer keybindings
        ])
        .split(area);

    render_header(frame, app, chunks[0]);
    render_filter_bar(frame, app, chunks[1]);
    render_log_list(frame, app, chunks[2]);
    render_footer(frame, app, chunks[3]);

    // Toast overlay (shown at bottom-center if present)
    if let Some(toast) = &app.toast {
        render_toast(frame, toast.is_error, &toast.message, area);
    }
}

fn render_header(frame: &mut Frame, app: &App, area: Rect) {
    let job_id_str = app
        .job_logs_job_id
        .map(|id| format!("Job #{}", id))
        .unwrap_or_else(|| "Job Logs".to_string());

    let task_str = app
        .job_logs_task
        .as_ref()
        .map(|t| {
            format!(
                " │ Task: {} │ Status: {}",
                t.start_time.get(..16).unwrap_or(&t.start_time),
                t.status
            )
        })
        .unwrap_or_default();

    let spinner_part = if app.job_logs_loading {
        let frame_idx = app.log_loading_state.spinner_frame % SPINNER_FRAMES.len();
        format!(" {} Loading…", SPINNER_FRAMES[frame_idx])
    } else {
        format!(" {} entries", app.job_logs.len())
    };

    let title = format!(" {} {}{}", job_id_str, task_str, spinner_part);

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .title_style(
            Style::default()
                .fg(Color::Cyan)
                .add_modifier(Modifier::BOLD),
        )
        .border_style(Style::default().fg(Color::Blue));

    frame.render_widget(block, area);
}

fn render_filter_bar(frame: &mut Frame, app: &App, area: Rect) {
    // Split into level filter (left) and search (right)
    let bar_chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Length(36), // level toggle buttons
            Constraint::Min(0),     // search input
        ])
        .split(area);

    // Level filter toggles
    let level_area = bar_chunks[0];
    let filter_block = Block::default()
        .borders(Borders::ALL)
        .title(" Level ")
        .border_style(Style::default().fg(Color::DarkGray));

    let inner = filter_block.inner(level_area);
    frame.render_widget(filter_block, level_area);

    // Render each level as a cell
    let levels = LogLevelFilter::ALL;
    let cell_w = inner.width / levels.len() as u16;
    for (i, level) in levels.iter().enumerate() {
        let cell = Rect {
            x: inner.x + i as u16 * cell_w,
            y: inner.y,
            width: cell_w,
            height: inner.height,
        };
        let is_active = *level == app.log_filter.level_filter;
        let style = if is_active {
            Style::default()
                .fg(Color::Black)
                .bg(Color::Yellow)
                .add_modifier(Modifier::BOLD)
        } else {
            Style::default().fg(Color::DarkGray)
        };
        let label = format!(" {} ", level.label());
        let p = Paragraph::new(label).style(style).alignment(Alignment::Center);
        frame.render_widget(p, cell);
    }

    // Search input
    let search_block = Block::default()
        .borders(Borders::ALL)
        .title(if app.log_filter.search_mode {
            " Search (Enter to apply, Esc to cancel) "
        } else {
            " Search (/ to activate) "
        })
        .border_style(if app.log_filter.search_mode {
            Style::default().fg(Color::Yellow)
        } else {
            Style::default().fg(Color::DarkGray)
        });

    let search_text = if app.log_filter.search_text.is_empty() && !app.log_filter.search_mode {
        Span::styled(
            "  type to filter…",
            Style::default().fg(Color::DarkGray),
        )
    } else {
        let display = format!(" {} ", app.log_filter.search_text);
        Span::styled(display, Style::default().fg(Color::White))
    };

    let search_para = Paragraph::new(Line::from(search_text)).block(search_block);
    frame.render_widget(search_para, bar_chunks[1]);
}

fn render_log_list(frame: &mut Frame, app: &App, area: Rect) {
    // Collect filtered entries
    let entries: Vec<(usize, &crate::olake::types::JobLogEntry)> = app
        .job_logs
        .iter()
        .enumerate()
        .filter(|(_, e)| {
            let level_ok = app.log_filter.level_filter.matches(&e.level);
            let search_ok = if app.log_filter.search_text.is_empty() {
                true
            } else {
                let q = app.log_filter.search_text.to_lowercase();
                e.message.to_lowercase().contains(&q) || e.level.to_lowercase().contains(&q)
            };
            level_ok && search_ok
        })
        .collect();

    let total = entries.len();

    // Build header info
    let pagination_hint = {
        let mut parts = Vec::new();
        if app.job_logs_has_more_older {
            parts.push("p: load older");
        }
        if app.job_logs_has_more_newer {
            parts.push("n: load newer");
        }
        if parts.is_empty() {
            String::new()
        } else {
            format!("  [{}]", parts.join(" | "))
        }
    };

    let title = if total == 0 {
        if app.job_logs_loading {
            " Loading logs… ".to_string()
        } else {
            " No logs (try changing filter) ".to_string()
        }
    } else {
        format!(" {} log entries{} ", total, pagination_hint)
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .border_style(Style::default().fg(Color::DarkGray));

    let inner = block.inner(area);
    frame.render_widget(block, area);

    if entries.is_empty() {
        if app.job_logs_loading {
            let frame_idx = app.log_loading_state.spinner_frame % SPINNER_FRAMES.len();
            let p = Paragraph::new(format!(
                "\n  {} Loading logs, please wait…",
                SPINNER_FRAMES[frame_idx]
            ))
            .style(Style::default().fg(Color::DarkGray));
            frame.render_widget(p, inner);
        } else {
            let p = Paragraph::new("\n  No log entries match the current filter.")
                .style(Style::default().fg(Color::DarkGray));
            frame.render_widget(p, inner);
        }
        return;
    }

    // Compute visible window
    let visible_height = inner.height as usize;
    let selected = app.selected_log_idx.min(total.saturating_sub(1));

    // Scroll so selected is always visible
    let scroll_offset = if selected < visible_height / 2 {
        0
    } else if selected + visible_height / 2 >= total {
        total.saturating_sub(visible_height)
    } else {
        selected - visible_height / 2
    };

    // Column widths (dynamic based on area width)
    let w = inner.width;
    let time_w = 20u16.min(w / 6);
    let level_w = 7u16;
    let msg_w = w.saturating_sub(time_w + level_w + 2); // 2 for separators

    // Render rows
    let items: Vec<ListItem> = entries
        .iter()
        .skip(scroll_offset)
        .take(visible_height)
        .enumerate()
        .map(|(vis_idx, (_orig_idx, entry))| {
            let filtered_pos = scroll_offset + vis_idx;
            let is_selected = filtered_pos == selected;

            let time_str = truncate(&entry.time, time_w as usize);
            let level_str = truncate(&entry.level.to_uppercase(), level_w as usize);
            let msg_str = truncate(&entry.message, msg_w as usize);

            let level_sty = level_style(&entry.level);

            let spans = if is_selected {
                vec![
                    Span::styled(
                        format!("{:<width$}", time_str, width = time_w as usize),
                        Style::default().fg(Color::Black).bg(Color::Cyan),
                    ),
                    Span::styled(
                        " ",
                        Style::default().fg(Color::Black).bg(Color::Cyan),
                    ),
                    Span::styled(
                        format!("{:<width$}", level_str, width = level_w as usize),
                        Style::default()
                            .fg(Color::Black)
                            .bg(Color::Cyan)
                            .add_modifier(Modifier::BOLD),
                    ),
                    Span::styled(
                        " ",
                        Style::default().fg(Color::Black).bg(Color::Cyan),
                    ),
                    Span::styled(
                        msg_str,
                        Style::default()
                            .fg(Color::Black)
                            .bg(Color::Cyan)
                            .add_modifier(Modifier::BOLD),
                    ),
                ]
            } else {
                vec![
                    Span::styled(
                        format!("{:<width$}", time_str, width = time_w as usize),
                        Style::default().fg(Color::DarkGray),
                    ),
                    Span::raw(" "),
                    Span::styled(
                        format!("{:<width$}", level_str, width = level_w as usize),
                        level_sty,
                    ),
                    Span::raw(" "),
                    Span::styled(msg_str, Style::default().fg(Color::White)),
                ]
            };

            ListItem::new(Line::from(spans))
        })
        .collect();

    let list = List::new(items);
    frame.render_widget(list, inner);
}

fn render_footer(frame: &mut Frame, app: &App, area: Rect) {
    let loading_hint = if app.job_logs_loading {
        "  ⠋ loading… "
    } else {
        ""
    };
    let text = format!(
        "{}q/Esc: close | j/k: scroll | PgUp/Dn: page | g/G: top/end | ←/→: level | /: search | d: download | r: refresh | p: older | n: newer",
        loading_hint
    );
    let footer = Paragraph::new(Line::from(Span::styled(
        text,
        Style::default().fg(Color::DarkGray),
    )));
    frame.render_widget(footer, area);
}

fn render_toast(frame: &mut Frame, is_error: bool, message: &str, area: Rect) {
    let toast_width = (message.len() as u16 + 6).min(area.width.saturating_sub(4));
    let toast_height = 3u16;
    let x = area.x + (area.width.saturating_sub(toast_width)) / 2;
    let y = area.y + area.height.saturating_sub(toast_height + 2);

    let toast_area = Rect {
        x,
        y,
        width: toast_width,
        height: toast_height,
    };

    let (border_color, prefix) = if is_error {
        (Color::Red, " ✗ ")
    } else {
        (Color::Green, " ✓ ")
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(border_color))
        .style(Style::default().bg(Color::Black));

    let text = format!("{}{}", prefix, message);
    let para = Paragraph::new(text)
        .block(block)
        .style(Style::default().fg(border_color))
        .alignment(Alignment::Center);

    frame.render_widget(Clear, toast_area);
    frame.render_widget(para, toast_area);
}
