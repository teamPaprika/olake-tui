mod app;
mod event;
mod olake;
mod ui;

use anyhow::Result;
use crossterm::{
    event::{DisableMouseCapture, EnableMouseCapture},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::prelude::*;
use tokio::sync::mpsc;

use crate::app::{Action, App, AppEvent};
use crate::olake::OlakeClient;

#[tokio::main]
async fn main() -> Result<()> {
    // --- OLake HTTP client ---------------------------------------------------
    // Base URL is read from OLAKE_API_URL env var (default: http://localhost:8000)
    let client = OlakeClient::new().map_err(|e| anyhow::anyhow!("Failed to create client: {e}"))?;

    // --- Channels ------------------------------------------------------------
    // action_tx: UI → background worker
    let (action_tx, _action_rx) = mpsc::unbounded_channel::<Action>();
    // app_event_tx / app_event_rx: background tasks → App
    let (app_event_tx, app_event_rx) = mpsc::unbounded_channel::<AppEvent>();

    // --- Build App -----------------------------------------------------------
    let mut app = App::new(client, action_tx, app_event_rx, app_event_tx);

    // --- Terminal setup ------------------------------------------------------
    enable_raw_mode()?;
    let mut stdout = std::io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    // --- Run -----------------------------------------------------------------
    let result = app.run(&mut terminal).await;

    // --- Restore terminal ----------------------------------------------------
    disable_raw_mode()?;
    execute!(
        terminal.backend_mut(),
        LeaveAlternateScreen,
        DisableMouseCapture
    )?;
    terminal.show_cursor()?;

    if let Err(e) = result {
        eprintln!("Error: {e:?}");
    }

    Ok(())
}
