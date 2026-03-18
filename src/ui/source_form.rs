use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::{App, SourceFormField, SourceType};

/// Render the source create/edit form.
pub fn render(frame: &mut Frame, area: Rect, app: &App) {
    // Center the form on screen
    let vchunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(2),
            Constraint::Min(0),
            Constraint::Length(1),
        ])
        .split(area);

    let hchunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(10),
            Constraint::Percentage(80),
            Constraint::Percentage(10),
        ])
        .split(vchunks[1]);

    let form_area = hchunks[1];

    let form = &app.source_form;
    let is_editing = app.source_form.editing_id.is_some();
    let title = if is_editing {
        " Edit Source "
    } else {
        " New Source "
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .title_style(Style::default().fg(Color::Green).add_modifier(Modifier::BOLD))
        .border_style(Style::default().fg(Color::Green));

    frame.render_widget(block.clone(), form_area);

    let inner = block.inner(form_area);

    // Build field list
    let fields = form.field_list();
    let total_fields = fields.len();

    // Layout: each field row = 3 lines (label + input + gap)
    // Plus 2 lines for source type selector
    // Plus 2 lines for buttons
    let mut constraints = vec![];
    // Source type selector row
    constraints.push(Constraint::Length(3));
    // Name field
    constraints.push(Constraint::Length(3));
    // Dynamic fields
    for _ in 0..form.dynamic_field_count() {
        constraints.push(Constraint::Length(3));
    }
    // Test connection result (if any)
    constraints.push(Constraint::Length(2));
    // Buttons
    constraints.push(Constraint::Length(3));
    // Rest
    constraints.push(Constraint::Min(0));

    let rows = Layout::default()
        .direction(Direction::Vertical)
        .margin(1)
        .constraints(constraints)
        .split(inner);

    let mut row_idx = 0;

    // ── Source Type Selector ──────────────────────────────────────────────
    let type_focused = matches!(form.focused_field, SourceFormField::SourceType);
    let type_label_style = if type_focused {
        Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    let type_block = Block::default()
        .borders(Borders::ALL)
        .title(" Source Type ")
        .border_style(if type_focused {
            Style::default().fg(Color::Yellow)
        } else {
            Style::default().fg(Color::DarkGray)
        });

    let type_options: Vec<Span> = SourceType::ALL
        .iter()
        .map(|t| {
            if *t == form.source_type {
                Span::styled(
                    format!(" [{}] ", t.label()),
                    Style::default()
                        .fg(Color::Black)
                        .bg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                )
            } else {
                Span::styled(
                    format!("  {}  ", t.label()),
                    Style::default().fg(Color::White),
                )
            }
        })
        .collect();

    let type_line = Line::from(type_options);
    let type_para = Paragraph::new(type_line).block(type_block);
    if row_idx < rows.len() {
        frame.render_widget(type_para, rows[row_idx]);
    }
    row_idx += 1;

    // ── Name Field ────────────────────────────────────────────────────────
    if row_idx < rows.len() {
        render_text_field(
            frame,
            rows[row_idx],
            "Name",
            &form.name,
            matches!(form.focused_field, SourceFormField::Name),
            false,
        );
    }
    row_idx += 1;

    // ── Dynamic Fields ────────────────────────────────────────────────────
    for field in &fields[2..] {
        // fields[0] = SourceType, fields[1] = Name; rest are dynamic
        if row_idx < rows.len() - 2 {
            let is_focused = form.focused_field == *field;
            let is_password = matches!(
                field,
                SourceFormField::Password | SourceFormField::SecretKey
            );
            let label = field.label();
            let value = form.get_field_value(field);
            render_text_field(frame, rows[row_idx], label, value, is_focused, is_password);
            row_idx += 1;
        }
    }

    // ── Connection test result ─────────────────────────────────────────────
    if row_idx < rows.len() - 1 {
        let result_text = if app.connection_testing {
            Line::from(Span::styled(
                "  ⠋ Testing connection…",
                Style::default().fg(Color::Yellow),
            ))
        } else if let Some(ref result) = app.connection_test_result {
            match result {
                Ok(msg) => Line::from(Span::styled(
                    format!("  ✓ {}", msg),
                    Style::default().fg(Color::Green),
                )),
                Err(msg) => Line::from(Span::styled(
                    format!("  ✗ {}", msg),
                    Style::default().fg(Color::Red),
                )),
            }
        } else {
            Line::from("")
        };
        frame.render_widget(Paragraph::new(result_text), rows[row_idx]);
        row_idx += 1;
    }

    // ── Buttons ───────────────────────────────────────────────────────────
    if row_idx < rows.len() {
        render_buttons(
            frame,
            rows[row_idx],
            &form.focused_field,
        );
    }

    // ── Status bar hint ───────────────────────────────────────────────────
    let hint = Paragraph::new(Line::from(Span::styled(
        " Tab/↑↓: navigate fields | ←/→: change type | Enter: confirm | Esc: cancel",
        Style::default().fg(Color::DarkGray),
    )));
    frame.render_widget(hint, vchunks[2]);
}

fn render_text_field(
    frame: &mut Frame,
    area: Rect,
    label: &str,
    value: &str,
    focused: bool,
    is_password: bool,
) {
    let border_style = if focused {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    let display = if is_password {
        "•".repeat(value.len())
    } else {
        value.to_string()
    };

    // Add cursor indicator when focused
    let content = if focused {
        format!("{}_", display)
    } else {
        display
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(format!(" {} ", label))
        .border_style(border_style);

    let para = Paragraph::new(content).block(block);
    frame.render_widget(para, area);
}

fn render_buttons(
    frame: &mut Frame,
    area: Rect,
    focused_field: &SourceFormField,
) {
    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(40),
            Constraint::Percentage(30),
            Constraint::Percentage(30),
        ])
        .split(area);

    let test_style = if matches!(focused_field, SourceFormField::TestButton) {
        Style::default().fg(Color::Black).bg(Color::Cyan).add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Cyan)
    };
    let test_block = Block::default()
        .borders(Borders::ALL)
        .border_style(if matches!(focused_field, SourceFormField::TestButton) {
            Style::default().fg(Color::Cyan)
        } else {
            Style::default().fg(Color::DarkGray)
        });
    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" ⚡ Test Connection", test_style))).block(test_block),
        chunks[0],
    );

    let save_style = if matches!(focused_field, SourceFormField::SaveButton) {
        Style::default().fg(Color::Black).bg(Color::Green).add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Green)
    };
    let save_block = Block::default()
        .borders(Borders::ALL)
        .border_style(if matches!(focused_field, SourceFormField::SaveButton) {
            Style::default().fg(Color::Green)
        } else {
            Style::default().fg(Color::DarkGray)
        });
    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" ✓ Save", save_style))).block(save_block),
        chunks[1],
    );

    let cancel_style = Style::default().fg(Color::Red);
    let cancel_block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::DarkGray));
    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" ✗ Cancel (Esc)", cancel_style)))
            .block(cancel_block),
        chunks[2],
    );
}
