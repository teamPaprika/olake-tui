use anyhow::Result;
use crossterm::event::{KeyCode, KeyModifiers};
use ratatui::prelude::*;
use std::time::Duration;
use tokio::sync::mpsc;

use crate::event::{self, TuiEvent};
use crate::olake::OlakeClient;
use crate::olake::types::{Entity, Job};
use crate::ui;

// ---------------------------------------------------------------------------
// Screen / Route
// ---------------------------------------------------------------------------

/// Which screen the app is currently showing.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Screen {
    /// Login screen — shown when not authenticated.
    Login,
    /// Main tabbed interface — shown when authenticated.
    Main,
}

/// Top-level tabs in the main view.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Tab {
    Dashboard,
    Sources,
    Destinations,
    Jobs,
}

impl Tab {
    pub const ALL: [Tab; 4] = [Tab::Dashboard, Tab::Sources, Tab::Destinations, Tab::Jobs];

    pub fn title(&self) -> &'static str {
        match self {
            Tab::Dashboard => "Dashboard",
            Tab::Sources => "Sources",
            Tab::Destinations => "Destinations",
            Tab::Jobs => "Jobs",
        }
    }
}

// ---------------------------------------------------------------------------
// Actions — UI → background worker
// ---------------------------------------------------------------------------

/// Actions triggered by the UI and dispatched to background workers.
#[derive(Debug, Clone)]
pub enum Action {
    /// Navigate to a specific screen.
    Navigate(Screen),
    /// Navigate to a tab within the main screen.
    SwitchTab(Tab),
    /// Attempt login with the provided credentials.
    Login { username: String, password: String },
    /// Load all sources from the API.
    LoadSources,
    /// Load all destinations from the API.
    LoadDestinations,
    /// Load all jobs from the API.
    LoadJobs,
    /// Refresh data for the current tab.
    RefreshCurrentTab,
    /// Select next item in list.
    SelectNext,
    /// Select previous item in list.
    SelectPrev,
    /// Create a new source (placeholder — details TBD).
    CreateSource,
    /// Test connection for a connector config (placeholder).
    TestConnection,
    /// Trigger a manual sync for the given job ID.
    Sync { job_id: String },
    /// Cancel the current in-flight operation.
    Cancel,
    /// Quit the application.
    Quit,
}

// ---------------------------------------------------------------------------
// AppEvent — background worker → UI
// ---------------------------------------------------------------------------

/// Events sent back from background tasks to the App.
#[derive(Debug)]
pub enum AppEvent {
    /// Login succeeded; provides the authenticated username.
    AuthSuccess { username: String },
    /// Login failed; provides an error message.
    AuthFailed { reason: String },
    /// Sources loaded from the API.
    SourcesLoaded(Vec<Entity>),
    /// Destinations loaded from the API.
    DestinationsLoaded(Vec<Entity>),
    /// Jobs loaded from the API.
    JobsLoaded(Vec<Job>),
    /// Connection test result.
    ConnectionTestResult { success: bool, message: String },
    /// A sync operation has been acknowledged by the server.
    SyncStarted { job_id: String },
    /// A generic error message to display to the user.
    Error(String),
}

// ---------------------------------------------------------------------------
// Loading states
// ---------------------------------------------------------------------------

/// Spinner frames for animated loading indicators.
const SPINNER_FRAMES: &[&str] = &["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

/// Tracks the state of an async operation.
#[derive(Debug, Default, Clone)]
pub struct LoadingState {
    pub loading: bool,
    pub spinner_frame: usize,
}

impl LoadingState {
    pub fn start(&mut self) {
        self.loading = true;
        self.spinner_frame = 0;
    }

    pub fn stop(&mut self) {
        self.loading = false;
    }

    pub fn tick(&mut self) {
        if self.loading {
            self.spinner_frame = (self.spinner_frame + 1) % SPINNER_FRAMES.len();
        }
    }

    pub fn spinner(&self) -> &'static str {
        SPINNER_FRAMES[self.spinner_frame % SPINNER_FRAMES.len()]
    }
}

// ---------------------------------------------------------------------------
// Auth state
// ---------------------------------------------------------------------------

#[derive(Debug, Default, Clone)]
pub struct AuthState {
    pub logged_in: bool,
    pub username: String,
}

// ---------------------------------------------------------------------------
// Login form state
// ---------------------------------------------------------------------------

#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub enum LoginField {
    #[default]
    Username,
    Password,
}

#[derive(Debug, Default, Clone)]
pub struct LoginForm {
    pub username: String,
    pub password: String,
    pub focused_field: LoginField,
    pub error: Option<String>,
    pub loading: LoadingState,
}

impl LoginForm {
    pub fn toggle_field(&mut self) {
        self.focused_field = match self.focused_field {
            LoginField::Username => LoginField::Password,
            LoginField::Password => LoginField::Username,
        };
    }

    pub fn push_char(&mut self, c: char) {
        match self.focused_field {
            LoginField::Username => self.username.push(c),
            LoginField::Password => self.password.push(c),
        }
    }

    pub fn pop_char(&mut self) {
        match self.focused_field {
            LoginField::Username => { self.username.pop(); }
            LoginField::Password => { self.password.pop(); }
        }
    }
}

// ---------------------------------------------------------------------------
// Application state
// ---------------------------------------------------------------------------

pub struct App {
    pub running: bool,
    pub screen: Screen,
    pub active_tab: Tab,
    pub auth: AuthState,
    pub login_form: LoginForm,

    /// Global error/notification message (shown in status bar).
    pub error_message: Option<String>,
    /// Success/info message (shown in status bar briefly).
    pub info_message: Option<String>,

    /// Loaded data
    pub sources: Vec<Entity>,
    pub destinations: Vec<Entity>,
    pub jobs: Vec<Job>,

    /// Selected row indices
    pub selected_source_idx: usize,
    pub selected_dest_idx: usize,
    pub selected_job_idx: usize,

    /// Loading state for various async operations.
    pub loading_sources: LoadingState,
    pub loading_destinations: LoadingState,
    pub loading_jobs: LoadingState,

    /// Sender for dispatching actions to the background worker.
    action_tx: mpsc::UnboundedSender<Action>,
    /// Receiver for events from background workers.
    event_rx: mpsc::UnboundedReceiver<AppEvent>,

    /// HTTP client (available for spawned tasks).
    pub client: OlakeClient,
    /// Sender given to background tasks so they can push events back.
    app_event_tx: mpsc::UnboundedSender<AppEvent>,
}

impl App {
    pub fn new(
        client: OlakeClient,
        action_tx: mpsc::UnboundedSender<Action>,
        event_rx: mpsc::UnboundedReceiver<AppEvent>,
        app_event_tx: mpsc::UnboundedSender<AppEvent>,
    ) -> Self {
        Self {
            running: true,
            screen: Screen::Login,
            active_tab: Tab::Dashboard,
            auth: AuthState::default(),
            login_form: LoginForm::default(),
            error_message: None,
            info_message: None,
            sources: Vec::new(),
            destinations: Vec::new(),
            jobs: Vec::new(),
            selected_source_idx: 0,
            selected_dest_idx: 0,
            selected_job_idx: 0,
            loading_sources: LoadingState::default(),
            loading_destinations: LoadingState::default(),
            loading_jobs: LoadingState::default(),
            action_tx,
            event_rx,
            client,
            app_event_tx,
        }
    }

    /// Convenience: dispatch an Action.
    pub fn dispatch(&self, action: Action) {
        let _ = self.action_tx.send(action);
    }

    // -----------------------------------------------------------------------
    // Main run loop
    // -----------------------------------------------------------------------

    pub async fn run<B: Backend>(&mut self, terminal: &mut Terminal<B>) -> Result<()>
    where
        B::Error: Send + Sync + 'static,
    {
        // Start the crossterm event loop.
        let (tui_tx, mut tui_rx) = mpsc::unbounded_channel::<TuiEvent>();
        event::start_event_loop(tui_tx, Duration::from_millis(100));

        while self.running {
            // Draw
            terminal.draw(|frame| ui::render(frame, self))?;

            // Process TUI events (non-blocking — drain what's ready)
            while let Ok(tui_event) = tui_rx.try_recv() {
                self.handle_tui_event(tui_event);
            }

            // Process AppEvents from background tasks (non-blocking)
            while let Ok(app_event) = self.event_rx.try_recv() {
                self.handle_app_event(app_event);
            }

            // Process pending actions
            // (Actions are sent to self.action_tx; we process them here inline
            //  because for now the "worker" runs in the same loop via spawned tasks.)
            // A brief yield lets tokio run any spawned background tasks.
            tokio::task::yield_now().await;
        }

        Ok(())
    }

    // -----------------------------------------------------------------------
    // TUI event handling
    // -----------------------------------------------------------------------

    fn handle_tui_event(&mut self, event: TuiEvent) {
        match event {
            TuiEvent::Tick => self.on_tick(),
            TuiEvent::Key(key) => self.handle_key(key),
            TuiEvent::Resize(_, _) => {}
            TuiEvent::Mouse(_) => {}
        }
    }

    fn on_tick(&mut self) {
        // Advance spinner frames
        self.login_form.loading.tick();
        self.loading_sources.tick();
        self.loading_destinations.tick();
        self.loading_jobs.tick();
    }

    fn handle_key(&mut self, key: crossterm::event::KeyEvent) {
        // Global quit
        if key.code == KeyCode::Char('c') && key.modifiers.contains(KeyModifiers::CONTROL) {
            self.running = false;
            return;
        }

        match &self.screen {
            Screen::Login => self.handle_login_key(key),
            Screen::Main => self.handle_main_key(key),
        }
    }

    fn handle_login_key(&mut self, key: crossterm::event::KeyEvent) {
        if self.login_form.loading.loading {
            // Only allow cancel during loading
            if key.code == KeyCode::Esc {
                self.login_form.loading.stop();
                self.login_form.error = Some("Login cancelled.".to_string());
            }
            return;
        }

        match key.code {
            KeyCode::Tab | KeyCode::Down => self.login_form.toggle_field(),
            KeyCode::BackTab | KeyCode::Up => self.login_form.toggle_field(),
            KeyCode::Enter => self.submit_login(),
            KeyCode::Char(c) => {
                self.login_form.error = None;
                self.login_form.push_char(c);
            }
            KeyCode::Backspace => self.login_form.pop_char(),
            KeyCode::Esc => self.running = false,
            _ => {}
        }
    }

    fn submit_login(&mut self) {
        let username = self.login_form.username.trim().to_string();
        let password = self.login_form.password.trim().to_string();

        if username.is_empty() {
            self.login_form.error = Some("Username is required.".to_string());
            self.login_form.focused_field = LoginField::Username;
            return;
        }
        if password.is_empty() {
            self.login_form.error = Some("Password is required.".to_string());
            self.login_form.focused_field = LoginField::Password;
            return;
        }

        self.login_form.loading.start();
        self.login_form.error = None;

        // Spawn background task for login
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        let uname = username.clone();
        let pwd = password.clone();
        tokio::spawn(async move {
            match client.login(uname.clone(), pwd).await {
                Ok(resp) => {
                    let _ = tx.send(AppEvent::AuthSuccess {
                        username: resp.username,
                    });
                }
                Err(e) => {
                    let reason: String = e.to_string();
                    let _ = tx.send(AppEvent::AuthFailed { reason });
                }
            }
        });
    }

    fn handle_main_key(&mut self, key: crossterm::event::KeyEvent) {
        match key.code {
            KeyCode::Char('q') => self.running = false,
            KeyCode::Char('1') => self.switch_to_tab(Tab::Dashboard),
            KeyCode::Char('2') => self.switch_to_tab(Tab::Sources),
            KeyCode::Char('3') => self.switch_to_tab(Tab::Destinations),
            KeyCode::Char('4') => self.switch_to_tab(Tab::Jobs),
            KeyCode::Tab => {
                let idx = Tab::ALL.iter().position(|t| *t == self.active_tab).unwrap_or(0);
                let new_tab = Tab::ALL[(idx + 1) % Tab::ALL.len()];
                self.switch_to_tab(new_tab);
            }
            KeyCode::BackTab => {
                let idx = Tab::ALL.iter().position(|t| *t == self.active_tab).unwrap_or(0);
                let new_tab = Tab::ALL[(idx + Tab::ALL.len() - 1) % Tab::ALL.len()];
                self.switch_to_tab(new_tab);
            }
            // List navigation
            KeyCode::Char('j') | KeyCode::Down => self.select_next(),
            KeyCode::Char('k') | KeyCode::Up => self.select_prev(),
            // Refresh
            KeyCode::Char('r') => self.refresh_current_tab(),
            _ => {}
        }
    }

    fn switch_to_tab(&mut self, tab: Tab) {
        self.active_tab = tab;
        // Auto-load data when switching to a data tab
        match tab {
            Tab::Sources => self.load_sources(),
            Tab::Destinations => self.load_destinations(),
            Tab::Jobs => self.load_jobs(),
            Tab::Dashboard => {}
        }
    }

    fn select_next(&mut self) {
        match self.active_tab {
            Tab::Sources => {
                if !self.sources.is_empty() {
                    self.selected_source_idx = (self.selected_source_idx + 1) % self.sources.len();
                }
            }
            Tab::Destinations => {
                if !self.destinations.is_empty() {
                    self.selected_dest_idx = (self.selected_dest_idx + 1) % self.destinations.len();
                }
            }
            Tab::Jobs => {
                if !self.jobs.is_empty() {
                    self.selected_job_idx = (self.selected_job_idx + 1) % self.jobs.len();
                }
            }
            Tab::Dashboard => {}
        }
    }

    fn select_prev(&mut self) {
        match self.active_tab {
            Tab::Sources => {
                if !self.sources.is_empty() {
                    self.selected_source_idx = self.selected_source_idx
                        .saturating_sub(1)
                        .min(self.sources.len() - 1);
                    if self.selected_source_idx == 0 && self.sources.len() > 0 {
                        // wrap around
                        // actually saturating_sub already handles 0 case; keep 0
                    }
                }
            }
            Tab::Destinations => {
                if !self.destinations.is_empty() {
                    self.selected_dest_idx = if self.selected_dest_idx == 0 {
                        self.destinations.len() - 1
                    } else {
                        self.selected_dest_idx - 1
                    };
                }
            }
            Tab::Jobs => {
                if !self.jobs.is_empty() {
                    self.selected_job_idx = if self.selected_job_idx == 0 {
                        self.jobs.len() - 1
                    } else {
                        self.selected_job_idx - 1
                    };
                }
            }
            Tab::Dashboard => {}
        }
    }

    fn refresh_current_tab(&mut self) {
        match self.active_tab {
            Tab::Sources => self.load_sources(),
            Tab::Destinations => self.load_destinations(),
            Tab::Jobs => self.load_jobs(),
            Tab::Dashboard => {
                self.load_sources();
                self.load_destinations();
                self.load_jobs();
            }
        }
    }

    fn load_sources(&mut self) {
        if self.loading_sources.loading {
            return;
        }
        self.loading_sources.start();
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        tokio::spawn(async move {
            match client.sources_list().await {
                Ok(data) => {
                    let _ = tx.send(AppEvent::SourcesLoaded(data));
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::Error(format!("Failed to load sources: {}", e)));
                }
            }
        });
    }

    fn load_destinations(&mut self) {
        if self.loading_destinations.loading {
            return;
        }
        self.loading_destinations.start();
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        tokio::spawn(async move {
            match client.destinations_list().await {
                Ok(data) => {
                    let _ = tx.send(AppEvent::DestinationsLoaded(data));
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::Error(format!("Failed to load destinations: {}", e)));
                }
            }
        });
    }

    fn load_jobs(&mut self) {
        if self.loading_jobs.loading {
            return;
        }
        self.loading_jobs.start();
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        tokio::spawn(async move {
            match client.jobs_list().await {
                Ok(data) => {
                    let _ = tx.send(AppEvent::JobsLoaded(data));
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::Error(format!("Failed to load jobs: {}", e)));
                }
            }
        });
    }

    // -----------------------------------------------------------------------
    // AppEvent handling
    // -----------------------------------------------------------------------

    fn handle_app_event(&mut self, event: AppEvent) {
        match event {
            AppEvent::AuthSuccess { username } => {
                self.login_form.loading.stop();
                self.auth.logged_in = true;
                self.auth.username = username;
                self.screen = Screen::Main;
                self.info_message = Some(format!("Welcome, {}!", self.auth.username));
            }
            AppEvent::AuthFailed { reason } => {
                self.login_form.loading.stop();
                self.login_form.error = Some(reason);
            }
            AppEvent::SourcesLoaded(data) => {
                self.loading_sources.stop();
                let count = data.len();
                self.sources = data;
                // clamp selected index
                if !self.sources.is_empty() && self.selected_source_idx >= self.sources.len() {
                    self.selected_source_idx = self.sources.len() - 1;
                }
                self.info_message = Some(format!("Loaded {} source(s).", count));
            }
            AppEvent::DestinationsLoaded(data) => {
                self.loading_destinations.stop();
                let count = data.len();
                self.destinations = data;
                if !self.destinations.is_empty() && self.selected_dest_idx >= self.destinations.len() {
                    self.selected_dest_idx = self.destinations.len() - 1;
                }
                self.info_message = Some(format!("Loaded {} destination(s).", count));
            }
            AppEvent::JobsLoaded(data) => {
                self.loading_jobs.stop();
                let count = data.len();
                self.jobs = data;
                if !self.jobs.is_empty() && self.selected_job_idx >= self.jobs.len() {
                    self.selected_job_idx = self.jobs.len() - 1;
                }
                self.info_message = Some(format!("Loaded {} job(s).", count));
            }
            AppEvent::ConnectionTestResult { success, message } => {
                if success {
                    self.info_message = Some(format!("✓ {}", message));
                } else {
                    self.error_message = Some(format!("✗ {}", message));
                }
            }
            AppEvent::SyncStarted { job_id } => {
                self.info_message = Some(format!("Sync started for job {}.", job_id));
            }
            AppEvent::Error(msg) => {
                self.error_message = Some(msg);
            }
        }
    }
}
