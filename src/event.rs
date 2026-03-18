use anyhow::Result;
use crossterm::event::{self, Event, KeyEvent, MouseEvent};
use std::time::Duration;
use tokio::sync::mpsc;

/// Unified event type consumed by the App.
#[derive(Debug)]
pub enum TuiEvent {
    /// A key press from the user.
    Key(KeyEvent),
    /// A mouse event.
    Mouse(MouseEvent),
    /// Terminal resize.
    Resize(u16, u16),
    /// Periodic tick for animations / polling.
    Tick,
}

/// Start the crossterm event listener in a tokio task.
/// Sends [`TuiEvent`]s on the provided sender.
pub fn start_event_loop(tx: mpsc::UnboundedSender<TuiEvent>, tick_rate: Duration) {
    tokio::spawn(async move {
        loop {
            // Convert tick_rate to a crossterm-compatible approach:
            // poll with a timeout equal to the tick rate.
            let timeout = tick_rate;
            let has_event = tokio::task::spawn_blocking(move || {
                event::poll(timeout)
            })
            .await;

            match has_event {
                Ok(Ok(true)) => {
                    // Read the event in a blocking task
                    let ev = tokio::task::spawn_blocking(|| event::read()).await;
                    match ev {
                        Ok(Ok(Event::Key(key))) => {
                            let _ = tx.send(TuiEvent::Key(key));
                        }
                        Ok(Ok(Event::Mouse(m))) => {
                            let _ = tx.send(TuiEvent::Mouse(m));
                        }
                        Ok(Ok(Event::Resize(w, h))) => {
                            let _ = tx.send(TuiEvent::Resize(w, h));
                        }
                        _ => {}
                    }
                }
                Ok(Ok(false)) => {
                    // Timeout — send tick
                    let _ = tx.send(TuiEvent::Tick);
                }
                _ => {
                    // Error polling — brief sleep then retry
                    tokio::time::sleep(Duration::from_millis(10)).await;
                }
            }
        }
    });
}

/// Simple blocking poll helper kept for compatibility / testing.
pub fn poll_event(timeout: Duration) -> Result<Option<Event>> {
    if event::poll(timeout)? {
        Ok(Some(event::read()?))
    } else {
        Ok(None)
    }
}
