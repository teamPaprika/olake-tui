use ratatui::prelude::*;
use ratatui::widgets::*;

use crate::app::{App, DestFormField, DestType};

/// Render the destination create/edit form.
pub fn render(frame: &mut Frame, area: Rect, app: &App) {
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

    let form = &app.dest_form;
    let is_editing = form.editing_id.is_some();
    let title = if is_editing {
        " Edit Destination "
    } else {
        " New Destination "
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(title)
        .title_style(
            Style::default()
                .fg(Color::Magenta)
                .add_modifier(Modifier::BOLD),
        )
        .border_style(Style::default().fg(Color::Magenta));

    frame.render_widget(block.clone(), form_area);
    let inner = block.inner(form_area);

    let fields = form.field_list();

    let mut constraints = vec![];
    constraints.push(Constraint::Length(3)); // dest type
    constraints.push(Constraint::Length(3)); // name
    for _ in 0..form.dynamic_field_count() {
        constraints.push(Constraint::Length(3));
    }
    constraints.push(Constraint::Length(2)); // test result
    constraints.push(Constraint::Length(3)); // buttons
    constraints.push(Constraint::Min(0));

    let rows = Layout::default()
        .direction(Direction::Vertical)
        .margin(1)
        .constraints(constraints)
        .split(inner);

    let mut row_idx = 0;

    // ── Destination Type Selector ─────────────────────────────────────────
    let type_focused = matches!(form.focused_field, DestFormField::DestType);
    let type_block = Block::default()
        .borders(Borders::ALL)
        .title(" Destination Type ")
        .border_style(if type_focused {
            Style::default().fg(Color::Yellow)
        } else {
            Style::default().fg(Color::DarkGray)
        });

    let type_options: Vec<Span> = DestType::ALL
        .iter()
        .map(|t| {
            if *t == form.dest_type {
                Span::styled(
                    format!(" [{}] ", t.label()),
                    Style::default()
                        .fg(Color::Black)
                        .bg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                )
            } else {
                Span::styled(format!("  {}  ", t.label()), Style::default().fg(Color::White))
            }
        })
        .collect();

    let type_para = Paragraph::new(Line::from(type_options)).block(type_block);
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
            matches!(form.focused_field, DestFormField::Name),
            false,
        );
    }
    row_idx += 1;

    // ── Dynamic Fields ────────────────────────────────────────────────────
    for field in &fields[2..] {
        if row_idx < rows.len() - 2 {
            let is_focused = form.focused_field == *field;
            let is_password = matches!(field, DestFormField::SecretKey | DestFormField::IcebergPassword);
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
        render_buttons(frame, rows[row_idx], &form.focused_field);
    }

    // ── Hint ─────────────────────────────────────────────────────────────
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

    let content = if focused {
        format!("{}_", display)
    } else {
        display
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .title(format!(" {} ", label))
        .border_style(border_style);

    frame.render_widget(Paragraph::new(content).block(block), area);
}

fn render_buttons(frame: &mut Frame, area: Rect, focused_field: &DestFormField) {
    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(40),
            Constraint::Percentage(30),
            Constraint::Percentage(30),
        ])
        .split(area);

    let test_style = if matches!(focused_field, DestFormField::TestButton) {
        Style::default()
            .fg(Color::Black)
            .bg(Color::Cyan)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Cyan)
    };
    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" ⚡ Test Connection", test_style))).block(
            Block::default().borders(Borders::ALL).border_style(
                if matches!(focused_field, DestFormField::TestButton) {
                    Style::default().fg(Color::Cyan)
                } else {
                    Style::default().fg(Color::DarkGray)
                },
            ),
        ),
        chunks[0],
    );

    let save_style = if matches!(focused_field, DestFormField::SaveButton) {
        Style::default()
            .fg(Color::Black)
            .bg(Color::Green)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::Green)
    };
    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(" ✓ Save", save_style))).block(
            Block::default().borders(Borders::ALL).border_style(
                if matches!(focused_field, DestFormField::SaveButton) {
                    Style::default().fg(Color::Green)
                } else {
                    Style::default().fg(Color::DarkGray)
                },
            ),
        ),
        chunks[1],
    );

    frame.render_widget(
        Paragraph::new(Line::from(Span::styled(
            " ✗ Cancel (Esc)",
            Style::default().fg(Color::Red),
        )))
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_style(Style::default().fg(Color::DarkGray)),
        ),
        chunks[2],
    );
}
