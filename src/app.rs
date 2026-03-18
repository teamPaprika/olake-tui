use anyhow::Result;
use crossterm::event::{KeyCode, KeyModifiers};
use ratatui::prelude::*;
use std::time::Duration;
use tokio::sync::mpsc;

use crate::event::{self, TuiEvent};
use crate::olake::OlakeClient;
use crate::olake::types::{
    Entity, EntityBase, EntityTestRequest, Job, JobLogEntry, JobTask,
    TaskLogsPaginationParams, TaskLogsDirection, TestConnectionStatus,
};
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
    /// Source create/edit form.
    SourceForm,
    /// Destination create/edit form.
    DestinationForm,
    /// Confirmation dialog (shown as overlay over current content).
    ConfirmDialog,
    /// Job logs viewer screen.
    JobLogs,
}

// ---------------------------------------------------------------------------
// Log direction
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LogDirection {
    Older,
    Newer,
}

// ---------------------------------------------------------------------------
// Log level filter
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LogLevelFilter {
    All,
    Info,
    Warn,
    Error,
}

impl LogLevelFilter {
    pub const ALL: [LogLevelFilter; 4] = [
        LogLevelFilter::All,
        LogLevelFilter::Info,
        LogLevelFilter::Warn,
        LogLevelFilter::Error,
    ];

    pub fn label(&self) -> &'static str {
        match self {
            LogLevelFilter::All => "All",
            LogLevelFilter::Info => "Info",
            LogLevelFilter::Warn => "Warn",
            LogLevelFilter::Error => "Error",
        }
    }

    pub fn next(&self) -> LogLevelFilter {
        let idx = Self::ALL.iter().position(|f| f == self).unwrap_or(0);
        Self::ALL[(idx + 1) % Self::ALL.len()]
    }

    pub fn prev(&self) -> LogLevelFilter {
        let idx = Self::ALL.iter().position(|f| f == self).unwrap_or(0);
        Self::ALL[(idx + Self::ALL.len() - 1) % Self::ALL.len()]
    }

    /// Returns true if the given log level string matches this filter.
    pub fn matches(&self, level: &str) -> bool {
        match self {
            LogLevelFilter::All => true,
            LogLevelFilter::Info => level.to_lowercase().contains("info"),
            LogLevelFilter::Warn => {
                let l = level.to_lowercase();
                l.contains("warn") || l.contains("warning")
            }
            LogLevelFilter::Error => {
                let l = level.to_lowercase();
                l.contains("error") || l.contains("fatal")
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Log filter state
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default)]
pub struct LogFilterState {
    pub search_text: String,
    pub level_filter: LogLevelFilter,
    pub search_mode: bool, // true when user is typing in the search box
}

impl Default for LogLevelFilter {
    fn default() -> Self {
        LogLevelFilter::All
    }
}

// ---------------------------------------------------------------------------
// Toast notification
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct Toast {
    pub message: String,
    pub is_error: bool,
    pub ticks_remaining: u32, // decrements on each tick; 0 = hidden
}

impl Toast {
    pub fn info(msg: impl Into<String>) -> Self {
        Toast {
            message: msg.into(),
            is_error: false,
            ticks_remaining: 30, // ~3 seconds at 10fps
        }
    }

    pub fn error(msg: impl Into<String>) -> Self {
        Toast {
            message: msg.into(),
            is_error: true,
            ticks_remaining: 40,
        }
    }
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
// Source type
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SourceType {
    Postgres,
    Mysql,
    MongoDB,
    Oracle,
    Mssql,
    Db2,
    Kafka,
    S3,
}

impl SourceType {
    pub const ALL: [SourceType; 8] = [
        SourceType::Postgres,
        SourceType::Mysql,
        SourceType::MongoDB,
        SourceType::Oracle,
        SourceType::Mssql,
        SourceType::Db2,
        SourceType::Kafka,
        SourceType::S3,
    ];

    pub fn label(&self) -> &'static str {
        match self {
            SourceType::Postgres => "PostgreSQL",
            SourceType::Mysql => "MySQL",
            SourceType::MongoDB => "MongoDB",
            SourceType::Oracle => "Oracle",
            SourceType::Mssql => "MSSQL",
            SourceType::Db2 => "DB2",
            SourceType::Kafka => "Kafka",
            SourceType::S3 => "S3",
        }
    }

    pub fn api_type(&self) -> &'static str {
        match self {
            SourceType::Postgres => "POSTGRES",
            SourceType::Mysql => "MYSQL",
            SourceType::MongoDB => "MONGODB",
            SourceType::Oracle => "ORACLE",
            SourceType::Mssql => "MSSQL",
            SourceType::Db2 => "DB2",
            SourceType::Kafka => "KAFKA",
            SourceType::S3 => "S3",
        }
    }

    pub fn next(&self) -> SourceType {
        let idx = Self::ALL.iter().position(|t| t == self).unwrap_or(0);
        Self::ALL[(idx + 1) % Self::ALL.len()]
    }

    pub fn prev(&self) -> SourceType {
        let idx = Self::ALL.iter().position(|t| t == self).unwrap_or(0);
        Self::ALL[(idx + Self::ALL.len() - 1) % Self::ALL.len()]
    }
}

// ---------------------------------------------------------------------------
// Destination type
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DestType {
    Iceberg,
    Parquet,
}

impl DestType {
    pub const ALL: [DestType; 2] = [DestType::Iceberg, DestType::Parquet];

    pub fn label(&self) -> &'static str {
        match self {
            DestType::Iceberg => "Apache Iceberg",
            DestType::Parquet => "Parquet",
        }
    }

    pub fn api_type(&self) -> &'static str {
        match self {
            DestType::Iceberg => "ICEBERG",
            DestType::Parquet => "PARQUET",
        }
    }

    pub fn next(&self) -> DestType {
        match self {
            DestType::Iceberg => DestType::Parquet,
            DestType::Parquet => DestType::Iceberg,
        }
    }

    pub fn prev(&self) -> DestType {
        self.next() // only 2 types
    }
}

// ---------------------------------------------------------------------------
// Source form fields
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SourceFormField {
    SourceType,
    Name,
    // Relational DB fields
    Host,
    Port,
    Username,
    Password,
    Database,
    // MongoDB extra fields
    AuthSource,
    ReplicaSet,
    ConnectionString,
    // Kafka fields
    BootstrapServers,
    Topic,
    GroupId,
    // S3 fields
    Bucket,
    Region,
    AccessKey,
    SecretKey,
    Prefix,
    // Buttons
    TestButton,
    SaveButton,
}

impl SourceFormField {
    pub fn label(&self) -> &'static str {
        match self {
            SourceFormField::SourceType => "Source Type",
            SourceFormField::Name => "Name",
            SourceFormField::Host => "Host",
            SourceFormField::Port => "Port",
            SourceFormField::Username => "Username",
            SourceFormField::Password => "Password",
            SourceFormField::Database => "Database",
            SourceFormField::AuthSource => "Auth Source",
            SourceFormField::ReplicaSet => "Replica Set",
            SourceFormField::ConnectionString => "Connection String",
            SourceFormField::BootstrapServers => "Bootstrap Servers",
            SourceFormField::Topic => "Topic",
            SourceFormField::GroupId => "Group ID",
            SourceFormField::Bucket => "Bucket",
            SourceFormField::Region => "Region",
            SourceFormField::AccessKey => "Access Key",
            SourceFormField::SecretKey => "Secret Key",
            SourceFormField::Prefix => "Prefix",
            SourceFormField::TestButton => "Test Connection",
            SourceFormField::SaveButton => "Save",
        }
    }
}

// ---------------------------------------------------------------------------
// Destination form fields
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DestFormField {
    DestType,
    Name,
    // Iceberg
    CatalogType,
    Warehouse,
    IcebergUsername,
    IcebergPassword,
    IcebergUri,
    // Parquet
    StorageType,
    Path,
    ParquetAccessKey,
    SecretKey,
    // Buttons
    TestButton,
    SaveButton,
}

impl DestFormField {
    pub fn label(&self) -> &'static str {
        match self {
            DestFormField::DestType => "Destination Type",
            DestFormField::Name => "Name",
            DestFormField::CatalogType => "Catalog Type (Glue/REST/Hive/JDBC)",
            DestFormField::Warehouse => "Warehouse Path",
            DestFormField::IcebergUsername => "Username",
            DestFormField::IcebergPassword => "Password",
            DestFormField::IcebergUri => "Catalog URI",
            DestFormField::StorageType => "Storage Type (S3/GCS/Local)",
            DestFormField::Path => "Path",
            DestFormField::ParquetAccessKey => "Access Key",
            DestFormField::SecretKey => "Secret Key",
            DestFormField::TestButton => "Test Connection",
            DestFormField::SaveButton => "Save",
        }
    }
}

// ---------------------------------------------------------------------------
// Source form state
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default)]
pub struct SourceFormState {
    pub editing_id: Option<i64>,
    pub source_type: SourceType,
    pub focused_field: SourceFormField,
    // Fields
    pub name: String,
    pub host: String,
    pub port: String,
    pub username: String,
    pub password: String,
    pub database: String,
    pub auth_source: String,
    pub replica_set: String,
    pub connection_string: String,
    pub bootstrap_servers: String,
    pub topic: String,
    pub group_id: String,
    pub bucket: String,
    pub region: String,
    pub access_key: String,
    pub secret_key: String,
    pub prefix: String,
}

impl Default for SourceType {
    fn default() -> Self {
        SourceType::Postgres
    }
}

impl Default for SourceFormField {
    fn default() -> Self {
        SourceFormField::SourceType
    }
}

impl SourceFormState {
    pub fn reset(&mut self) {
        *self = SourceFormState::default();
    }

    pub fn populate_from_entity(&mut self, entity: &Entity) {
        use serde_json::Value;

        self.editing_id = Some(entity.id);
        self.name = entity.name.clone();

        // Try to map the entity_type back to SourceType
        self.source_type = match entity.entity_type.to_uppercase().as_str() {
            "POSTGRES" | "POSTGRESQL" => SourceType::Postgres,
            "MYSQL" => SourceType::Mysql,
            "MONGODB" | "MONGO" => SourceType::MongoDB,
            "ORACLE" => SourceType::Oracle,
            "MSSQL" => SourceType::Mssql,
            "DB2" => SourceType::Db2,
            "KAFKA" => SourceType::Kafka,
            "S3" => SourceType::S3,
            _ => SourceType::Postgres,
        };

        // Parse config JSON
        let config: Value = if let Value::String(s) = &entity.config {
            serde_json::from_str(s).unwrap_or(Value::Object(Default::default()))
        } else {
            entity.config.clone()
        };

        let get = |key: &str| -> String {
            config.get(key).and_then(|v| v.as_str()).unwrap_or("").to_string()
        };

        self.host = get("host");
        self.port = get("port");
        self.username = get("username");
        self.password = get("password");
        self.database = get("database");
        self.auth_source = get("auth_source");
        self.replica_set = get("replica_set");
        self.connection_string = get("connection_string");
        self.bootstrap_servers = get("bootstrap_servers");
        self.topic = get("topic");
        self.group_id = get("group_id");
        self.bucket = get("bucket");
        self.region = get("region");
        self.access_key = get("access_key");
        self.secret_key = get("secret_key");
        self.prefix = get("prefix");

        self.focused_field = SourceFormField::Name;
    }

    /// Returns the ordered list of fields for the current source type.
    pub fn field_list(&self) -> Vec<SourceFormField> {
        let mut fields = vec![SourceFormField::SourceType, SourceFormField::Name];
        match self.source_type {
            SourceType::MongoDB => {
                fields.push(SourceFormField::Host);
                fields.push(SourceFormField::Port);
                fields.push(SourceFormField::Username);
                fields.push(SourceFormField::Password);
                fields.push(SourceFormField::Database);
                fields.push(SourceFormField::AuthSource);
                fields.push(SourceFormField::ReplicaSet);
            }
            SourceType::Kafka => {
                fields.push(SourceFormField::BootstrapServers);
                fields.push(SourceFormField::Topic);
                fields.push(SourceFormField::GroupId);
            }
            SourceType::S3 => {
                fields.push(SourceFormField::Bucket);
                fields.push(SourceFormField::Region);
                fields.push(SourceFormField::AccessKey);
                fields.push(SourceFormField::SecretKey);
                fields.push(SourceFormField::Prefix);
            }
            _ => {
                // Relational DBs
                fields.push(SourceFormField::Host);
                fields.push(SourceFormField::Port);
                fields.push(SourceFormField::Username);
                fields.push(SourceFormField::Password);
                fields.push(SourceFormField::Database);
            }
        }
        fields.push(SourceFormField::TestButton);
        fields.push(SourceFormField::SaveButton);
        fields
    }

    pub fn dynamic_field_count(&self) -> usize {
        // total fields minus SourceType, Name, TestButton, SaveButton
        self.field_list().len().saturating_sub(4)
    }

    pub fn get_field_value(&self, field: &SourceFormField) -> &str {
        match field {
            SourceFormField::Name => &self.name,
            SourceFormField::Host => &self.host,
            SourceFormField::Port => &self.port,
            SourceFormField::Username => &self.username,
            SourceFormField::Password => &self.password,
            SourceFormField::Database => &self.database,
            SourceFormField::AuthSource => &self.auth_source,
            SourceFormField::ReplicaSet => &self.replica_set,
            SourceFormField::ConnectionString => &self.connection_string,
            SourceFormField::BootstrapServers => &self.bootstrap_servers,
            SourceFormField::Topic => &self.topic,
            SourceFormField::GroupId => &self.group_id,
            SourceFormField::Bucket => &self.bucket,
            SourceFormField::Region => &self.region,
            SourceFormField::AccessKey => &self.access_key,
            SourceFormField::SecretKey => &self.secret_key,
            SourceFormField::Prefix => &self.prefix,
            _ => "",
        }
    }

    pub fn get_field_value_mut(&mut self, field: &SourceFormField) -> Option<&mut String> {
        match field {
            SourceFormField::Name => Some(&mut self.name),
            SourceFormField::Host => Some(&mut self.host),
            SourceFormField::Port => Some(&mut self.port),
            SourceFormField::Username => Some(&mut self.username),
            SourceFormField::Password => Some(&mut self.password),
            SourceFormField::Database => Some(&mut self.database),
            SourceFormField::AuthSource => Some(&mut self.auth_source),
            SourceFormField::ReplicaSet => Some(&mut self.replica_set),
            SourceFormField::ConnectionString => Some(&mut self.connection_string),
            SourceFormField::BootstrapServers => Some(&mut self.bootstrap_servers),
            SourceFormField::Topic => Some(&mut self.topic),
            SourceFormField::GroupId => Some(&mut self.group_id),
            SourceFormField::Bucket => Some(&mut self.bucket),
            SourceFormField::Region => Some(&mut self.region),
            SourceFormField::AccessKey => Some(&mut self.access_key),
            SourceFormField::SecretKey => Some(&mut self.secret_key),
            SourceFormField::Prefix => Some(&mut self.prefix),
            _ => None,
        }
    }

    pub fn nav_next(&mut self) {
        let fields = self.field_list();
        let idx = fields.iter().position(|f| f == &self.focused_field).unwrap_or(0);
        self.focused_field = fields[(idx + 1) % fields.len()].clone();
    }

    pub fn nav_prev(&mut self) {
        let fields = self.field_list();
        let idx = fields.iter().position(|f| f == &self.focused_field).unwrap_or(0);
        self.focused_field = fields[(idx + fields.len() - 1) % fields.len()].clone();
    }

    /// Build the JSON config string for API submission.
    pub fn build_config(&self) -> String {
        let mut map = serde_json::Map::new();
        match self.source_type {
            SourceType::MongoDB => {
                map.insert("host".into(), self.host.clone().into());
                map.insert("port".into(), self.port.clone().into());
                map.insert("username".into(), self.username.clone().into());
                map.insert("password".into(), self.password.clone().into());
                map.insert("database".into(), self.database.clone().into());
                if !self.auth_source.is_empty() {
                    map.insert("auth_source".into(), self.auth_source.clone().into());
                }
                if !self.replica_set.is_empty() {
                    map.insert("replica_set".into(), self.replica_set.clone().into());
                }
            }
            SourceType::Kafka => {
                map.insert("bootstrap_servers".into(), self.bootstrap_servers.clone().into());
                map.insert("topic".into(), self.topic.clone().into());
                map.insert("group_id".into(), self.group_id.clone().into());
            }
            SourceType::S3 => {
                map.insert("bucket".into(), self.bucket.clone().into());
                map.insert("region".into(), self.region.clone().into());
                map.insert("access_key".into(), self.access_key.clone().into());
                map.insert("secret_key".into(), self.secret_key.clone().into());
                if !self.prefix.is_empty() {
                    map.insert("prefix".into(), self.prefix.clone().into());
                }
            }
            _ => {
                map.insert("host".into(), self.host.clone().into());
                map.insert("port".into(), self.port.clone().into());
                map.insert("username".into(), self.username.clone().into());
                map.insert("password".into(), self.password.clone().into());
                map.insert("database".into(), self.database.clone().into());
            }
        }
        serde_json::to_string(&map).unwrap_or_else(|_| "{}".to_string())
    }
}

// ---------------------------------------------------------------------------
// Destination form state
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Default)]
pub struct DestFormState {
    pub editing_id: Option<i64>,
    pub dest_type: DestType,
    pub focused_field: DestFormField,
    // Fields
    pub name: String,
    pub catalog_type: String,
    pub warehouse: String,
    pub iceberg_username: String,
    pub iceberg_password: String,
    pub iceberg_uri: String,
    pub storage_type: String,
    pub path: String,
    pub parquet_access_key: String,
    pub secret_key: String,
}

impl Default for DestType {
    fn default() -> Self {
        DestType::Iceberg
    }
}

impl Default for DestFormField {
    fn default() -> Self {
        DestFormField::DestType
    }
}

impl DestFormState {
    pub fn reset(&mut self) {
        *self = DestFormState::default();
    }

    pub fn populate_from_entity(&mut self, entity: &Entity) {
        use serde_json::Value;

        self.editing_id = Some(entity.id);
        self.name = entity.name.clone();

        self.dest_type = match entity.entity_type.to_uppercase().as_str() {
            "ICEBERG" | "APACHE_ICEBERG" => DestType::Iceberg,
            "PARQUET" => DestType::Parquet,
            _ => DestType::Iceberg,
        };

        let config: Value = if let Value::String(s) = &entity.config {
            serde_json::from_str(s).unwrap_or(Value::Object(Default::default()))
        } else {
            entity.config.clone()
        };

        let get = |key: &str| -> String {
            config.get(key).and_then(|v| v.as_str()).unwrap_or("").to_string()
        };

        self.catalog_type = get("catalog_type");
        self.warehouse = get("warehouse");
        self.iceberg_username = get("username");
        self.iceberg_password = get("password");
        self.iceberg_uri = get("uri");
        self.storage_type = get("storage_type");
        self.path = get("path");
        self.parquet_access_key = get("access_key");
        self.secret_key = get("secret_key");

        self.focused_field = DestFormField::Name;
    }

    pub fn field_list(&self) -> Vec<DestFormField> {
        let mut fields = vec![DestFormField::DestType, DestFormField::Name];
        match self.dest_type {
            DestType::Iceberg => {
                fields.push(DestFormField::CatalogType);
                fields.push(DestFormField::Warehouse);
                fields.push(DestFormField::IcebergUri);
                fields.push(DestFormField::IcebergUsername);
                fields.push(DestFormField::IcebergPassword);
            }
            DestType::Parquet => {
                fields.push(DestFormField::StorageType);
                fields.push(DestFormField::Path);
                fields.push(DestFormField::ParquetAccessKey);
                fields.push(DestFormField::SecretKey);
            }
        }
        fields.push(DestFormField::TestButton);
        fields.push(DestFormField::SaveButton);
        fields
    }

    pub fn dynamic_field_count(&self) -> usize {
        self.field_list().len().saturating_sub(4)
    }

    pub fn get_field_value(&self, field: &DestFormField) -> &str {
        match field {
            DestFormField::Name => &self.name,
            DestFormField::CatalogType => &self.catalog_type,
            DestFormField::Warehouse => &self.warehouse,
            DestFormField::IcebergUsername => &self.iceberg_username,
            DestFormField::IcebergPassword => &self.iceberg_password,
            DestFormField::IcebergUri => &self.iceberg_uri,
            DestFormField::StorageType => &self.storage_type,
            DestFormField::Path => &self.path,
            DestFormField::ParquetAccessKey => &self.parquet_access_key,
            DestFormField::SecretKey => &self.secret_key,
            _ => "",
        }
    }

    pub fn get_field_value_mut(&mut self, field: &DestFormField) -> Option<&mut String> {
        match field {
            DestFormField::Name => Some(&mut self.name),
            DestFormField::CatalogType => Some(&mut self.catalog_type),
            DestFormField::Warehouse => Some(&mut self.warehouse),
            DestFormField::IcebergUsername => Some(&mut self.iceberg_username),
            DestFormField::IcebergPassword => Some(&mut self.iceberg_password),
            DestFormField::IcebergUri => Some(&mut self.iceberg_uri),
            DestFormField::StorageType => Some(&mut self.storage_type),
            DestFormField::Path => Some(&mut self.path),
            DestFormField::ParquetAccessKey => Some(&mut self.parquet_access_key),
            DestFormField::SecretKey => Some(&mut self.secret_key),
            _ => None,
        }
    }

    pub fn nav_next(&mut self) {
        let fields = self.field_list();
        let idx = fields.iter().position(|f| f == &self.focused_field).unwrap_or(0);
        self.focused_field = fields[(idx + 1) % fields.len()].clone();
    }

    pub fn nav_prev(&mut self) {
        let fields = self.field_list();
        let idx = fields.iter().position(|f| f == &self.focused_field).unwrap_or(0);
        self.focused_field = fields[(idx + fields.len() - 1) % fields.len()].clone();
    }

    pub fn build_config(&self) -> String {
        let mut map = serde_json::Map::new();
        match self.dest_type {
            DestType::Iceberg => {
                if !self.catalog_type.is_empty() {
                    map.insert("catalog_type".into(), self.catalog_type.clone().into());
                }
                map.insert("warehouse".into(), self.warehouse.clone().into());
                if !self.iceberg_uri.is_empty() {
                    map.insert("uri".into(), self.iceberg_uri.clone().into());
                }
                if !self.iceberg_username.is_empty() {
                    map.insert("username".into(), self.iceberg_username.clone().into());
                }
                if !self.iceberg_password.is_empty() {
                    map.insert("password".into(), self.iceberg_password.clone().into());
                }
            }
            DestType::Parquet => {
                if !self.storage_type.is_empty() {
                    map.insert("storage_type".into(), self.storage_type.clone().into());
                }
                map.insert("path".into(), self.path.clone().into());
                if !self.parquet_access_key.is_empty() {
                    map.insert("access_key".into(), self.parquet_access_key.clone().into());
                }
                if !self.secret_key.is_empty() {
                    map.insert("secret_key".into(), self.secret_key.clone().into());
                }
            }
        }
        serde_json::to_string(&map).unwrap_or_else(|_| "{}".to_string())
    }
}

// ---------------------------------------------------------------------------
// Confirmation dialog state
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub enum ConfirmAction {
    DeleteSource(i64),
    DeleteDestination(i64),
}

#[derive(Debug, Clone, Default)]
pub struct ConfirmDialogState {
    pub title: String,
    pub message: String,
    pub yes_selected: bool,
    pub on_confirm: Option<ConfirmAction>,
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
    // -----------------------------------------------------------------------
    // Job logs actions
    // -----------------------------------------------------------------------
    /// Open the job logs screen for the given job ID (and optional task ID).
    OpenJobLogs { job_id: i64, task: JobTask },
    /// Load logs in the given direction (pagination).
    LoadJobLogs { direction: LogDirection },
    /// Set log search text.
    SearchLogs(String),
    /// Set log level filter.
    SetLogLevel(LogLevelFilter),
    /// Download logs for the current job to a local file.
    DownloadLogs,
    /// Close the logs screen (return to main).
    CloseLogs,
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

    // CRUD events
    SourceCreated(Entity),
    SourceUpdated(Entity),
    SourceDeleted(String),
    DestinationCreated(Entity),
    DestinationUpdated(Entity),
    DestinationDeleted(String),
    ConnectionTestSuccess(String),
    ConnectionTestFailed(String),

    // Job logs events
    /// Log entries loaded; includes direction so the handler can prepend/append.
    JobLogsLoaded {
        entries: Vec<JobLogEntry>,
        older_cursor: i64,
        newer_cursor: i64,
        has_more_older: bool,
        has_more_newer: bool,
        direction: LogDirection,
    },
    /// An error occurred while loading logs.
    JobLogsError(String),
    /// Log file has been saved locally; contains the path.
    LogDownloadReady(String),
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
            LoginField::Username => {
                self.username.pop();
            }
            LoginField::Password => {
                self.password.pop();
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Application state
// ---------------------------------------------------------------------------

pub struct App {
    pub running: bool,
    pub screen: Screen,
    /// The tab that was active before opening a form (used for overlay/return)
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

    // CRUD form state
    pub source_form: SourceFormState,
    pub dest_form: DestFormState,
    pub confirm_dialog: ConfirmDialogState,
    pub connection_test_result: Option<Result<String, String>>,
    pub connection_testing: bool,

    // -----------------------------------------------------------------------
    // Job logs state
    // -----------------------------------------------------------------------
    /// All log entries currently displayed.
    pub job_logs: Vec<JobLogEntry>,
    /// Cursor returned by the last request (used for next page).
    pub job_logs_older_cursor: i64,
    pub job_logs_newer_cursor: i64,
    pub job_logs_has_more_older: bool,
    pub job_logs_has_more_newer: bool,
    /// Whether logs are currently being fetched.
    pub job_logs_loading: bool,
    /// Index of the currently selected log entry.
    pub selected_log_idx: usize,
    /// Log filter state (search + level).
    pub log_filter: LogFilterState,
    /// The job ID whose logs are displayed.
    pub job_logs_job_id: Option<i64>,
    /// The task currently being viewed (contains file_path etc.).
    pub job_logs_task: Option<JobTask>,
    /// Spinner for log loading animation.
    pub log_loading_state: LoadingState,

    // -----------------------------------------------------------------------
    // Toast notification
    // -----------------------------------------------------------------------
    pub toast: Option<Toast>,

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
            source_form: SourceFormState::default(),
            dest_form: DestFormState::default(),
            confirm_dialog: ConfirmDialogState::default(),
            connection_test_result: None,
            connection_testing: false,
            // Job logs
            job_logs: Vec::new(),
            job_logs_older_cursor: -1,
            job_logs_newer_cursor: -1,
            job_logs_has_more_older: false,
            job_logs_has_more_newer: false,
            job_logs_loading: false,
            selected_log_idx: 0,
            log_filter: LogFilterState::default(),
            job_logs_job_id: None,
            job_logs_task: None,
            log_loading_state: LoadingState::default(),
            toast: None,
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
        self.login_form.loading.tick();
        self.loading_sources.tick();
        self.loading_destinations.tick();
        self.loading_jobs.tick();
        self.log_loading_state.tick();

        // Tick down toast
        if let Some(toast) = &mut self.toast {
            if toast.ticks_remaining > 0 {
                toast.ticks_remaining -= 1;
            } else {
                self.toast = None;
            }
        }
    }

    fn handle_key(&mut self, key: crossterm::event::KeyEvent) {
        // Global quit
        if key.code == KeyCode::Char('c') && key.modifiers.contains(KeyModifiers::CONTROL) {
            self.running = false;
            return;
        }

        match self.screen.clone() {
            Screen::Login => self.handle_login_key(key),
            Screen::Main => self.handle_main_key(key),
            Screen::SourceForm => self.handle_source_form_key(key),
            Screen::DestinationForm => self.handle_dest_form_key(key),
            Screen::ConfirmDialog => self.handle_confirm_dialog_key(key),
            Screen::JobLogs => self.handle_job_logs_key(key),
        }
    }

    fn handle_login_key(&mut self, key: crossterm::event::KeyEvent) {
        if self.login_form.loading.loading {
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
            KeyCode::Char('j') | KeyCode::Down => self.select_next(),
            KeyCode::Char('k') | KeyCode::Up => self.select_prev(),
            KeyCode::Char('r') => self.refresh_current_tab(),

            // CRUD actions per tab
            KeyCode::Char('a') => match self.active_tab {
                Tab::Sources => self.open_source_form(None),
                Tab::Destinations => self.open_dest_form(None),
                _ => {}
            },
            KeyCode::Char('e') => match self.active_tab {
                Tab::Sources => {
                    if let Some(entity) = self.sources.get(self.selected_source_idx).cloned() {
                        self.open_source_form(Some(entity));
                    }
                }
                Tab::Destinations => {
                    if let Some(entity) = self.destinations.get(self.selected_dest_idx).cloned() {
                        self.open_dest_form(Some(entity));
                    }
                }
                _ => {}
            },
            KeyCode::Char('d') => match self.active_tab {
                Tab::Sources => {
                    if let Some(entity) = self.sources.get(self.selected_source_idx).cloned() {
                        self.open_confirm_delete_source(entity.id, &entity.name.clone());
                    }
                }
                Tab::Destinations => {
                    if let Some(entity) = self.destinations.get(self.selected_dest_idx).cloned() {
                        self.open_confirm_delete_dest(entity.id, &entity.name.clone());
                    }
                }
                _ => {}
            },
            KeyCode::Char('t') => match self.active_tab {
                Tab::Sources => {
                    if let Some(entity) = self.sources.get(self.selected_source_idx).cloned() {
                        self.test_source_connection_for_entity(&entity);
                    }
                }
                Tab::Destinations => {
                    if let Some(entity) = self.destinations.get(self.selected_dest_idx).cloned() {
                        self.test_dest_connection_for_entity(&entity);
                    }
                }
                _ => {}
            },
            KeyCode::Char('l') => {
                if self.active_tab == Tab::Jobs {
                    self.open_job_logs_for_selected();
                }
            }
            _ => {}
        }
    }

    // ── Source form key handler ────────────────────────────────────────────

    fn handle_source_form_key(&mut self, key: crossterm::event::KeyEvent) {
        match key.code {
            KeyCode::Esc => {
                self.screen = Screen::Main;
                self.connection_test_result = None;
                self.connection_testing = false;
            }
            KeyCode::Tab | KeyCode::Down => {
                self.source_form.nav_next();
            }
            KeyCode::BackTab | KeyCode::Up => {
                self.source_form.nav_prev();
            }
            KeyCode::Left => {
                if matches!(self.source_form.focused_field, SourceFormField::SourceType) {
                    self.source_form.source_type = self.source_form.source_type.prev();
                    // Reset to first field after type change
                    self.source_form.focused_field = SourceFormField::SourceType;
                }
            }
            KeyCode::Right => {
                if matches!(self.source_form.focused_field, SourceFormField::SourceType) {
                    self.source_form.source_type = self.source_form.source_type.next();
                    self.source_form.focused_field = SourceFormField::SourceType;
                }
            }
            KeyCode::Enter => {
                let focused = self.source_form.focused_field.clone();
                match focused {
                    SourceFormField::TestButton => self.submit_source_test(),
                    SourceFormField::SaveButton => self.submit_source_save(),
                    _ => self.source_form.nav_next(),
                }
            }
            KeyCode::Backspace => {
                let focused = self.source_form.focused_field.clone();
                if let Some(val) = self.source_form.get_field_value_mut(&focused) {
                    val.pop();
                }
            }
            KeyCode::Char(c) => {
                let focused = self.source_form.focused_field.clone();
                if let Some(val) = self.source_form.get_field_value_mut(&focused) {
                    val.push(c);
                }
            }
            _ => {}
        }
    }

    // ── Destination form key handler ──────────────────────────────────────

    fn handle_dest_form_key(&mut self, key: crossterm::event::KeyEvent) {
        match key.code {
            KeyCode::Esc => {
                self.screen = Screen::Main;
                self.connection_test_result = None;
                self.connection_testing = false;
            }
            KeyCode::Tab | KeyCode::Down => {
                self.dest_form.nav_next();
            }
            KeyCode::BackTab | KeyCode::Up => {
                self.dest_form.nav_prev();
            }
            KeyCode::Left => {
                if matches!(self.dest_form.focused_field, DestFormField::DestType) {
                    self.dest_form.dest_type = self.dest_form.dest_type.prev();
                    self.dest_form.focused_field = DestFormField::DestType;
                }
            }
            KeyCode::Right => {
                if matches!(self.dest_form.focused_field, DestFormField::DestType) {
                    self.dest_form.dest_type = self.dest_form.dest_type.next();
                    self.dest_form.focused_field = DestFormField::DestType;
                }
            }
            KeyCode::Enter => {
                let focused = self.dest_form.focused_field.clone();
                match focused {
                    DestFormField::TestButton => self.submit_dest_test(),
                    DestFormField::SaveButton => self.submit_dest_save(),
                    _ => self.dest_form.nav_next(),
                }
            }
            KeyCode::Backspace => {
                let focused = self.dest_form.focused_field.clone();
                if let Some(val) = self.dest_form.get_field_value_mut(&focused) {
                    val.pop();
                }
            }
            KeyCode::Char(c) => {
                let focused = self.dest_form.focused_field.clone();
                if let Some(val) = self.dest_form.get_field_value_mut(&focused) {
                    val.push(c);
                }
            }
            _ => {}
        }
    }

    // ── Confirm dialog key handler ────────────────────────────────────────

    fn handle_confirm_dialog_key(&mut self, key: crossterm::event::KeyEvent) {
        match key.code {
            KeyCode::Esc | KeyCode::Char('n') | KeyCode::Char('N') => {
                self.screen = Screen::Main;
                self.confirm_dialog = ConfirmDialogState::default();
            }
            KeyCode::Left | KeyCode::Right => {
                self.confirm_dialog.yes_selected = !self.confirm_dialog.yes_selected;
            }
            KeyCode::Char('y') | KeyCode::Char('Y') => {
                self.confirm_dialog.yes_selected = true;
                self.execute_confirm();
            }
            KeyCode::Enter => {
                if self.confirm_dialog.yes_selected {
                    self.execute_confirm();
                } else {
                    self.screen = Screen::Main;
                    self.confirm_dialog = ConfirmDialogState::default();
                }
            }
            _ => {}
        }
    }

    fn execute_confirm(&mut self) {
        if let Some(action) = self.confirm_dialog.on_confirm.take() {
            match action {
                ConfirmAction::DeleteSource(id) => self.delete_source(id),
                ConfirmAction::DeleteDestination(id) => self.delete_destination(id),
            }
        }
        self.screen = Screen::Main;
        self.confirm_dialog = ConfirmDialogState::default();
    }

    // -----------------------------------------------------------------------
    // Navigation helpers
    // -----------------------------------------------------------------------

    fn open_source_form(&mut self, entity: Option<Entity>) {
        self.source_form.reset();
        self.connection_test_result = None;
        self.connection_testing = false;
        if let Some(e) = entity {
            self.source_form.populate_from_entity(&e);
        }
        self.screen = Screen::SourceForm;
    }

    fn open_dest_form(&mut self, entity: Option<Entity>) {
        self.dest_form.reset();
        self.connection_test_result = None;
        self.connection_testing = false;
        if let Some(e) = entity {
            self.dest_form.populate_from_entity(&e);
        }
        self.screen = Screen::DestinationForm;
    }

    fn open_confirm_delete_source(&mut self, id: i64, name: &str) {
        self.confirm_dialog = ConfirmDialogState {
            title: "Delete Source".to_string(),
            message: format!("Delete source '{}'? This cannot be undone.", name),
            yes_selected: false,
            on_confirm: Some(ConfirmAction::DeleteSource(id)),
        };
        self.screen = Screen::ConfirmDialog;
    }

    fn open_confirm_delete_dest(&mut self, id: i64, name: &str) {
        self.confirm_dialog = ConfirmDialogState {
            title: "Delete Destination".to_string(),
            message: format!("Delete destination '{}'? This cannot be undone.", name),
            yes_selected: false,
            on_confirm: Some(ConfirmAction::DeleteDestination(id)),
        };
        self.screen = Screen::ConfirmDialog;
    }

    fn switch_to_tab(&mut self, tab: Tab) {
        self.active_tab = tab;
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
                    self.selected_source_idx = if self.selected_source_idx == 0 {
                        self.sources.len() - 1
                    } else {
                        self.selected_source_idx - 1
                    };
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

    // -----------------------------------------------------------------------
    // Data loaders
    // -----------------------------------------------------------------------

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
                    let _ = tx.send(AppEvent::Error(format!(
                        "Failed to load destinations: {}",
                        e
                    )));
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
    // Source CRUD actions
    // -----------------------------------------------------------------------

    fn submit_source_save(&mut self) {
        let form = &self.source_form;
        if form.name.trim().is_empty() {
            self.error_message = Some("Source name is required.".to_string());
            return;
        }

        let entity_base = EntityBase {
            name: form.name.trim().to_string(),
            entity_type: form.source_type.api_type().to_string(),
            version: "latest".to_string(),
            config: form.build_config(),
        };

        let editing_id = form.editing_id;
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            if let Some(id) = editing_id {
                match client.sources_update(id, entity_base).await {
                    Ok(entity) => {
                        let _ = tx.send(AppEvent::SourceUpdated(entity));
                    }
                    Err(e) => {
                        let _ = tx.send(AppEvent::Error(format!("Failed to update source: {}", e)));
                    }
                }
            } else {
                match client.sources_create(entity_base).await {
                    Ok(_) => {
                        // Reload sources list; create returns EntityBase, not Entity
                        match client.sources_list().await {
                            Ok(list) => {
                                let _ = tx.send(AppEvent::SourcesLoaded(list));
                            }
                            Err(e) => {
                                let _ = tx.send(AppEvent::Error(format!("Source created but failed to reload: {}", e)));
                            }
                        }
                    }
                    Err(e) => {
                        let _ = tx.send(AppEvent::Error(format!("Failed to create source: {}", e)));
                    }
                }
            }
        });

        self.screen = Screen::Main;
    }

    fn submit_source_test(&mut self) {
        if self.connection_testing {
            return;
        }
        self.connection_testing = true;
        self.connection_test_result = None;

        let form = &self.source_form;
        let req = EntityTestRequest {
            entity_type: form.source_type.api_type().to_string(),
            version: "latest".to_string(),
            config: form.build_config(),
            source_type: None,
            source_version: None,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            match client.sources_test_connection(req).await {
                Ok(resp) => {
                    let success = resp.connection_result.status == TestConnectionStatus::Succeeded;
                    if success {
                        let _ = tx.send(AppEvent::ConnectionTestSuccess(
                            resp.connection_result.message,
                        ));
                    } else {
                        let _ = tx.send(AppEvent::ConnectionTestFailed(
                            resp.connection_result.message,
                        ));
                    }
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::ConnectionTestFailed(e.to_string()));
                }
            }
        });
    }

    fn delete_source(&mut self, id: i64) {
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        tokio::spawn(async move {
            match client.sources_delete(id).await {
                Ok(resp) => {
                    let _ = tx.send(AppEvent::SourceDeleted(resp.name));
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::Error(format!("Failed to delete source: {}", e)));
                }
            }
        });
    }

    fn test_source_connection_for_entity(&mut self, entity: &Entity) {
        if self.connection_testing {
            return;
        }
        self.connection_testing = true;
        self.connection_test_result = None;

        let config_str = match &entity.config {
            serde_json::Value::String(s) => s.clone(),
            other => serde_json::to_string(other).unwrap_or_else(|_| "{}".to_string()),
        };

        let req = EntityTestRequest {
            entity_type: entity.entity_type.clone(),
            version: entity.version.clone(),
            config: config_str,
            source_type: None,
            source_version: None,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            match client.sources_test_connection(req).await {
                Ok(resp) => {
                    let success = resp.connection_result.status == TestConnectionStatus::Succeeded;
                    if success {
                        let _ = tx.send(AppEvent::ConnectionTestSuccess(
                            resp.connection_result.message,
                        ));
                    } else {
                        let _ = tx.send(AppEvent::ConnectionTestFailed(
                            resp.connection_result.message,
                        ));
                    }
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::ConnectionTestFailed(e.to_string()));
                }
            }
        });
    }

    // -----------------------------------------------------------------------
    // Destination CRUD actions
    // -----------------------------------------------------------------------

    fn submit_dest_save(&mut self) {
        let form = &self.dest_form;
        if form.name.trim().is_empty() {
            self.error_message = Some("Destination name is required.".to_string());
            return;
        }

        let entity_base = EntityBase {
            name: form.name.trim().to_string(),
            entity_type: form.dest_type.api_type().to_string(),
            version: "latest".to_string(),
            config: form.build_config(),
        };

        let editing_id = form.editing_id;
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            if let Some(id) = editing_id {
                match client.destinations_update(id, entity_base).await {
                    Ok(_) => {
                        // destinations_update returns EntityBase; reload list
                        match client.destinations_list().await {
                            Ok(list) => {
                                let _ = tx.send(AppEvent::DestinationsLoaded(list));
                            }
                            Err(e) => {
                                let _ = tx.send(AppEvent::Error(format!(
                                    "Destination updated but failed to reload: {}",
                                    e
                                )));
                            }
                        }
                    }
                    Err(e) => {
                        let _ = tx.send(AppEvent::Error(format!(
                            "Failed to update destination: {}",
                            e
                        )));
                    }
                }
            } else {
                match client.destinations_create(entity_base).await {
                    Ok(_) => match client.destinations_list().await {
                        Ok(list) => {
                            let _ = tx.send(AppEvent::DestinationsLoaded(list));
                        }
                        Err(e) => {
                            let _ = tx.send(AppEvent::Error(format!(
                                "Destination created but failed to reload: {}",
                                e
                            )));
                        }
                    },
                    Err(e) => {
                        let _ = tx.send(AppEvent::Error(format!(
                            "Failed to create destination: {}",
                            e
                        )));
                    }
                }
            }
        });

        self.screen = Screen::Main;
    }

    fn submit_dest_test(&mut self) {
        if self.connection_testing {
            return;
        }
        self.connection_testing = true;
        self.connection_test_result = None;

        let form = &self.dest_form;
        let req = EntityTestRequest {
            entity_type: form.dest_type.api_type().to_string(),
            version: "latest".to_string(),
            config: form.build_config(),
            source_type: None,
            source_version: None,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            match client.destinations_test_connection(req).await {
                Ok(resp) => {
                    let success = resp.connection_result.status == TestConnectionStatus::Succeeded;
                    if success {
                        let _ = tx.send(AppEvent::ConnectionTestSuccess(
                            resp.connection_result.message,
                        ));
                    } else {
                        let _ = tx.send(AppEvent::ConnectionTestFailed(
                            resp.connection_result.message,
                        ));
                    }
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::ConnectionTestFailed(e.to_string()));
                }
            }
        });
    }

    fn delete_destination(&mut self, id: i64) {
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        tokio::spawn(async move {
            match client.destinations_delete(id).await {
                Ok(resp) => {
                    let _ = tx.send(AppEvent::DestinationDeleted(resp.name));
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::Error(format!(
                        "Failed to delete destination: {}",
                        e
                    )));
                }
            }
        });
    }

    fn test_dest_connection_for_entity(&mut self, entity: &Entity) {
        if self.connection_testing {
            return;
        }
        self.connection_testing = true;
        self.connection_test_result = None;

        let config_str = match &entity.config {
            serde_json::Value::String(s) => s.clone(),
            other => serde_json::to_string(other).unwrap_or_else(|_| "{}".to_string()),
        };

        let req = EntityTestRequest {
            entity_type: entity.entity_type.clone(),
            version: entity.version.clone(),
            config: config_str,
            source_type: None,
            source_version: None,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            match client.destinations_test_connection(req).await {
                Ok(resp) => {
                    let success = resp.connection_result.status == TestConnectionStatus::Succeeded;
                    if success {
                        let _ = tx.send(AppEvent::ConnectionTestSuccess(
                            resp.connection_result.message,
                        ));
                    } else {
                        let _ = tx.send(AppEvent::ConnectionTestFailed(
                            resp.connection_result.message,
                        ));
                    }
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::ConnectionTestFailed(e.to_string()));
                }
            }
        });
    }

    // -----------------------------------------------------------------------
    // Job logs key handler
    // -----------------------------------------------------------------------

    fn handle_job_logs_key(&mut self, key: crossterm::event::KeyEvent) {
        // If in search mode, handle text input
        if self.log_filter.search_mode {
            match key.code {
                KeyCode::Esc => {
                    self.log_filter.search_mode = false;
                }
                KeyCode::Enter => {
                    self.log_filter.search_mode = false;
                    self.selected_log_idx = 0;
                }
                KeyCode::Backspace => {
                    self.log_filter.search_text.pop();
                }
                KeyCode::Char(c) => {
                    self.log_filter.search_text.push(c);
                }
                _ => {}
            }
            return;
        }

        match key.code {
            KeyCode::Char('q') | KeyCode::Esc => {
                self.close_logs();
            }
            KeyCode::Char('j') | KeyCode::Down => {
                self.log_select_next();
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.log_select_prev();
            }
            KeyCode::PageDown => {
                for _ in 0..20 {
                    self.log_select_next();
                }
            }
            KeyCode::PageUp => {
                for _ in 0..20 {
                    self.log_select_prev();
                }
            }
            KeyCode::Char('g') => {
                self.selected_log_idx = 0;
            }
            KeyCode::Char('G') => {
                let visible = self.filtered_logs_count();
                if visible > 0 {
                    self.selected_log_idx = visible - 1;
                }
            }
            KeyCode::Right => {
                self.log_filter.level_filter = self.log_filter.level_filter.next();
                self.selected_log_idx = 0;
            }
            KeyCode::Left => {
                self.log_filter.level_filter = self.log_filter.level_filter.prev();
                self.selected_log_idx = 0;
            }
            KeyCode::Char('/') => {
                self.log_filter.search_mode = true;
            }
            KeyCode::Char('d') => {
                self.download_logs();
            }
            KeyCode::Char('r') => {
                // Refresh: reload from start
                self.job_logs.clear();
                self.job_logs_older_cursor = -1;
                self.job_logs_newer_cursor = -1;
                self.selected_log_idx = 0;
                self.load_job_logs(LogDirection::Older);
            }
            KeyCode::Char('n') => {
                // Load newer logs
                if self.job_logs_has_more_newer && !self.job_logs_loading {
                    self.load_job_logs(LogDirection::Newer);
                }
            }
            KeyCode::Char('p') => {
                // Load older logs
                if self.job_logs_has_more_older && !self.job_logs_loading {
                    self.load_job_logs(LogDirection::Older);
                }
            }
            _ => {}
        }
    }

    fn filtered_logs_count(&self) -> usize {
        self.job_logs
            .iter()
            .filter(|e| {
                let level_ok = self.log_filter.level_filter.matches(&e.level);
                let search_ok = if self.log_filter.search_text.is_empty() {
                    true
                } else {
                    let q = self.log_filter.search_text.to_lowercase();
                    e.message.to_lowercase().contains(&q)
                        || e.level.to_lowercase().contains(&q)
                };
                level_ok && search_ok
            })
            .count()
    }

    fn log_select_next(&mut self) {
        let count = self.filtered_logs_count();
        if count > 0 {
            self.selected_log_idx = (self.selected_log_idx + 1).min(count - 1);
        }
    }

    fn log_select_prev(&mut self) {
        self.selected_log_idx = self.selected_log_idx.saturating_sub(1);
    }

    fn close_logs(&mut self) {
        self.screen = Screen::Main;
        self.job_logs.clear();
        self.job_logs_loading = false;
        self.job_logs_job_id = None;
        self.job_logs_task = None;
        self.selected_log_idx = 0;
        self.log_filter = LogFilterState::default();
    }

    // -----------------------------------------------------------------------
    // Open job logs from Jobs tab
    // -----------------------------------------------------------------------

    fn open_job_logs_for_selected(&mut self) {
        if self.jobs.is_empty() {
            return;
        }
        let job = match self.jobs.get(self.selected_job_idx) {
            Some(j) => j.clone(),
            None => return,
        };

        // Fetch tasks first, then open logs
        let job_id = job.id;
        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        // Reset log state
        self.job_logs.clear();
        self.job_logs_older_cursor = -1;
        self.job_logs_newer_cursor = -1;
        self.job_logs_has_more_older = false;
        self.job_logs_has_more_newer = false;
        self.selected_log_idx = 0;
        self.log_filter = LogFilterState::default();
        self.job_logs_job_id = Some(job_id);
        self.job_logs_task = None;
        self.job_logs_loading = true;
        self.log_loading_state.start();
        self.screen = Screen::JobLogs;

        // Spawn: fetch tasks → pick most recent → load logs
        tokio::spawn(async move {
            let tasks = match client.jobs_get_tasks(job_id).await {
                Ok(t) => t,
                Err(e) => {
                    let _ = tx.send(AppEvent::JobLogsError(format!(
                        "Failed to fetch tasks: {}",
                        e
                    )));
                    return;
                }
            };

            // Pick the most recent task (last in list)
            let task = match tasks.into_iter().last() {
                Some(t) => t,
                None => {
                    let _ = tx.send(AppEvent::JobLogsError(
                        "No tasks found for this job.".to_string(),
                    ));
                    return;
                }
            };

            let file_path = task.file_path.clone();
            let params = TaskLogsPaginationParams {
                cursor: -1,
                limit: 1000,
                direction: TaskLogsDirection::Older,
            };

            let task_id = task.runtime.clone(); // runtime is used as task ID
            match client
                .jobs_get_logs(job_id, &task_id, params, &file_path)
                .await
            {
                Ok(resp) => {
                    let entries: Vec<JobLogEntry> = resp
                        .logs
                        .into_iter()
                        .map(|l| JobLogEntry {
                            level: l.level,
                            time: l.time,
                            message: l.message,
                        })
                        .collect();
                    let _ = tx.send(AppEvent::JobLogsLoaded {
                        entries,
                        older_cursor: resp.older_cursor,
                        newer_cursor: resp.newer_cursor,
                        has_more_older: resp.has_more_older,
                        has_more_newer: resp.has_more_newer,
                        direction: LogDirection::Older,
                    });
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::JobLogsError(format!(
                        "Failed to load logs: {}",
                        e
                    )));
                }
            }
        });
    }

    // -----------------------------------------------------------------------
    // Load job logs (pagination)
    // -----------------------------------------------------------------------

    fn load_job_logs(&mut self, direction: LogDirection) {
        if self.job_logs_loading {
            return;
        }
        let job_id = match self.job_logs_job_id {
            Some(id) => id,
            None => return,
        };
        let task = match &self.job_logs_task {
            Some(t) => t.clone(),
            None => return,
        };

        self.job_logs_loading = true;
        self.log_loading_state.start();

        let cursor = match direction {
            LogDirection::Older => self.job_logs_older_cursor,
            LogDirection::Newer => self.job_logs_newer_cursor,
        };

        let api_direction = match direction {
            LogDirection::Older => TaskLogsDirection::Older,
            LogDirection::Newer => TaskLogsDirection::Newer,
        };

        let params = TaskLogsPaginationParams {
            cursor,
            limit: 500,
            direction: api_direction,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();
        let task_id = task.runtime.clone();
        let file_path = task.file_path.clone();

        tokio::spawn(async move {
            match client
                .jobs_get_logs(job_id, &task_id, params, &file_path)
                .await
            {
                Ok(resp) => {
                    let entries: Vec<JobLogEntry> = resp
                        .logs
                        .into_iter()
                        .map(|l| JobLogEntry {
                            level: l.level,
                            time: l.time,
                            message: l.message,
                        })
                        .collect();
                    let _ = tx.send(AppEvent::JobLogsLoaded {
                        entries,
                        older_cursor: resp.older_cursor,
                        newer_cursor: resp.newer_cursor,
                        has_more_older: resp.has_more_older,
                        has_more_newer: resp.has_more_newer,
                        direction,
                    });
                }
                Err(e) => {
                    let _ = tx.send(AppEvent::JobLogsError(format!(
                        "Failed to load logs: {}",
                        e
                    )));
                }
            }
        });
    }

    // -----------------------------------------------------------------------
    // Download logs
    // -----------------------------------------------------------------------

    fn download_logs(&mut self) {
        let job_id = match self.job_logs_job_id {
            Some(id) => id,
            None => return,
        };
        let task_id = match &self.job_logs_task {
            Some(t) => t.runtime.clone(),
            None => return,
        };

        let client = self.client.clone();
        let tx = self.app_event_tx.clone();

        tokio::spawn(async move {
            match client.jobs_download_logs(job_id, &task_id).await {
                Ok(bytes) => {
                    // Save to ~/.local/share/olake/logs/<job_id>-<timestamp>.log
                    let home = std::env::var("HOME").unwrap_or_else(|_| ".".to_string());
                    let log_dir =
                        std::path::PathBuf::from(format!("{}/.local/share/olake/logs", home));
                    let _ = tokio::fs::create_dir_all(&log_dir).await;
                    let ts = chrono_timestamp();
                    let filename = format!("{}-{}.tar.gz", job_id, ts);
                    let path = log_dir.join(&filename);
                    match tokio::fs::write(&path, &bytes).await {
                        Ok(_) => {
                            let path_str = path.to_string_lossy().to_string();
                            let _ = tx.send(AppEvent::LogDownloadReady(path_str));
                        }
                        Err(e) => {
                            let _ =
                                tx.send(AppEvent::JobLogsError(format!("Save failed: {}", e)));
                        }
                    }
                }
                Err(e) => {
                    let _ =
                        tx.send(AppEvent::JobLogsError(format!("Download failed: {}", e)));
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
                if !self.sources.is_empty() && self.selected_source_idx >= self.sources.len() {
                    self.selected_source_idx = self.sources.len() - 1;
                }
                self.info_message = Some(format!("Loaded {} source(s).", count));
            }
            AppEvent::DestinationsLoaded(data) => {
                self.loading_destinations.stop();
                let count = data.len();
                self.destinations = data;
                if !self.destinations.is_empty()
                    && self.selected_dest_idx >= self.destinations.len()
                {
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
                self.connection_testing = false;
                if success {
                    self.connection_test_result = Some(Ok(message.clone()));
                    self.info_message = Some(format!("✓ {}", message));
                } else {
                    self.connection_test_result = Some(Err(message.clone()));
                    self.error_message = Some(format!("✗ {}", message));
                }
            }
            AppEvent::SyncStarted { job_id } => {
                self.info_message = Some(format!("Sync started for job {}.", job_id));
            }
            AppEvent::Error(msg) => {
                self.error_message = Some(msg);
            }

            // CRUD events
            AppEvent::SourceCreated(entity) => {
                self.info_message = Some(format!("Source '{}' created.", entity.name));
                self.load_sources();
            }
            AppEvent::SourceUpdated(entity) => {
                self.info_message = Some(format!("Source '{}' updated.", entity.name));
                // Update in-place
                if let Some(existing) = self.sources.iter_mut().find(|e| e.id == entity.id) {
                    *existing = entity;
                } else {
                    self.load_sources();
                }
            }
            AppEvent::SourceDeleted(name) => {
                self.info_message = Some(format!("Source '{}' deleted.", name));
                self.load_sources();
            }
            AppEvent::DestinationCreated(entity) => {
                self.info_message = Some(format!("Destination '{}' created.", entity.name));
                self.load_destinations();
            }
            AppEvent::DestinationUpdated(entity) => {
                self.info_message = Some(format!("Destination '{}' updated.", entity.name));
                if let Some(existing) = self.destinations.iter_mut().find(|e| e.id == entity.id) {
                    *existing = entity;
                } else {
                    self.load_destinations();
                }
            }
            AppEvent::DestinationDeleted(name) => {
                self.info_message = Some(format!("Destination '{}' deleted.", name));
                self.load_destinations();
            }
            AppEvent::ConnectionTestSuccess(msg) => {
                self.connection_testing = false;
                self.connection_test_result = Some(Ok(msg.clone()));
                self.info_message = Some(format!("✓ Connection succeeded: {}", msg));
            }
            AppEvent::ConnectionTestFailed(msg) => {
                self.connection_testing = false;
                self.connection_test_result = Some(Err(msg.clone()));
                self.error_message = Some(format!("✗ Connection failed: {}", msg));
            }

            // Job logs events
            AppEvent::JobLogsLoaded {
                entries,
                older_cursor,
                newer_cursor,
                has_more_older,
                has_more_newer,
                direction,
            } => {
                self.job_logs_loading = false;
                self.log_loading_state.stop();
                self.job_logs_older_cursor = older_cursor;
                self.job_logs_newer_cursor = newer_cursor;
                self.job_logs_has_more_older = has_more_older;
                self.job_logs_has_more_newer = has_more_newer;
                match direction {
                    LogDirection::Older => {
                        // Prepend older entries
                        let mut new_logs = entries;
                        new_logs.extend(self.job_logs.drain(..));
                        self.job_logs = new_logs;
                        // Adjust selection to account for prepended entries
                    }
                    LogDirection::Newer => {
                        // Append newer entries
                        self.job_logs.extend(entries);
                    }
                }
                // Clamp selection
                let count = self.filtered_logs_count();
                if count > 0 && self.selected_log_idx >= count {
                    self.selected_log_idx = count - 1;
                }
            }
            AppEvent::JobLogsError(msg) => {
                self.job_logs_loading = false;
                self.log_loading_state.stop();
                // If we're on the logs screen but have no task, go back to main
                if self.job_logs_task.is_none() && self.screen == Screen::JobLogs {
                    self.screen = Screen::Main;
                    self.toast = Some(Toast::error(format!("Logs: {}", msg)));
                } else {
                    self.toast = Some(Toast::error(format!("Logs error: {}", msg)));
                }
            }
            AppEvent::LogDownloadReady(path) => {
                self.toast = Some(Toast::info(format!("Logs saved: {}", path)));
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Simple timestamp helper (no chrono dep — just uses SystemTime)
// ---------------------------------------------------------------------------

fn chrono_timestamp() -> u64 {
    use std::time::{SystemTime, UNIX_EPOCH};
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}
