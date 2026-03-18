use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// ---------------------------------------------------------------------------
// Common / Shared
// ---------------------------------------------------------------------------

/// Standard API response envelope.
#[derive(Debug, Deserialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub message: String,
    pub data: T,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct UnknownObject(pub serde_json::Value);

// ---------------------------------------------------------------------------
// Auth Types
// ---------------------------------------------------------------------------

#[derive(Debug, Serialize)]
pub struct LoginRequest {
    pub username: String,
    pub password: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct LoginResponse {
    pub username: String,
}

#[derive(Debug, Serialize)]
pub struct SignupRequest {
    pub email: String,
    pub username: String,
    pub password: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SignupResponse {
    pub email: String,
    pub username: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct AuthCheckResponse {
    pub username: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct TelemetryIdResponse {
    pub user_id: String,
    pub version: String,
}

// ---------------------------------------------------------------------------
// Log Entries
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogEntry {
    pub level: String,
    pub time: String,
    pub message: String,
}

// ---------------------------------------------------------------------------
// Entity (Source / Destination)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EntityJob {
    pub id: i64,
    pub name: String,
    pub job_name: String,
    pub activate: bool,
    #[serde(default)]
    pub last_run_time: String,
    #[serde(default)]
    pub last_run_state: String,
    #[serde(default)]
    pub destination_name: Option<String>,
    #[serde(default)]
    pub destination_type: Option<String>,
    #[serde(default)]
    pub source_name: Option<String>,
    #[serde(default)]
    pub source_type: Option<String>,
}

/// Full entity (source or destination) as returned from GET endpoints.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Entity {
    pub id: i64,
    pub name: String,
    #[serde(rename = "type")]
    pub entity_type: String,
    pub version: String,
    /// Config as a JSON string (or already parsed object from server).
    pub config: serde_json::Value,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
    #[serde(default)]
    pub created_by: String,
    #[serde(default)]
    pub updated_by: String,
    #[serde(default)]
    pub jobs: Vec<EntityJob>,
}

/// Request body for create/update.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EntityBase {
    pub name: String,
    #[serde(rename = "type")]
    pub entity_type: String,
    pub version: String,
    /// JSON string (stringified config object).
    pub config: String,
}

/// Request body for connection test.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EntityTestRequest {
    #[serde(rename = "type")]
    pub entity_type: String,
    pub version: String,
    pub config: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_version: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ConnectionResult {
    pub message: String,
    pub status: TestConnectionStatus,
}

#[derive(Debug, Clone, Deserialize)]
pub struct EntityTestResponse {
    pub connection_result: ConnectionResult,
    #[serde(default)]
    pub logs: Vec<LogEntry>,
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum TestConnectionStatus {
    Failed,
    Succeeded,
}

#[derive(Debug, Clone, Deserialize)]
pub struct VersionsResponse {
    pub version: Vec<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SpecResponse {
    #[serde(rename = "type")]
    pub spec_type: String,
    pub version: String,
    pub spec: serde_json::Value,
}

/// Request for spec endpoint.
#[derive(Debug, Clone, Serialize)]
pub struct SpecRequest {
    #[serde(rename = "type")]
    pub entity_type: String,
    pub version: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_version: Option<String>,
}

// ---------------------------------------------------------------------------
// Stream / Discover Types
// ---------------------------------------------------------------------------

/// Sync mode for a stream.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SyncMode {
    FullRefresh,
    Incremental,
    Cdc,
    StrictCdc,
}

impl std::fmt::Display for SyncMode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SyncMode::FullRefresh => write!(f, "Full Refresh"),
            SyncMode::Incremental => write!(f, "Incremental"),
            SyncMode::Cdc => write!(f, "CDC"),
            SyncMode::StrictCdc => write!(f, "Strict CDC"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DefaultStreamProperties {
    pub normalization: bool,
    pub append_mode: bool,
}

/// A single stream definition as returned from discover.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Stream {
    pub name: String,
    #[serde(default)]
    pub namespace: Option<String>,
    #[serde(default)]
    pub json_schema: Option<serde_json::Value>,
    #[serde(default)]
    pub supported_sync_modes: Vec<String>,
    #[serde(default)]
    pub source_defined_cursor: Option<bool>,
    #[serde(default)]
    pub default_cursor_field: Vec<String>,
    #[serde(default)]
    pub available_cursor_fields: Vec<String>,
    #[serde(default)]
    pub source_defined_primary_key: Vec<String>,
    #[serde(default)]
    pub destination_database: Option<String>,
    #[serde(default)]
    pub destination_table: Option<String>,
    #[serde(default)]
    pub default_stream_properties: DefaultStreamProperties,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StreamData {
    pub sync_mode: SyncMode,
    #[serde(default)]
    pub skip_nested_flattening: Option<bool>,
    #[serde(default)]
    pub cursor_field: Vec<String>,
    #[serde(default)]
    pub destination_sync_mode: String,
    #[serde(default)]
    pub sort_key: Option<Vec<String>>,
    pub stream: Stream,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SelectedColumns {
    pub columns: Vec<String>,
    pub sync_new_columns: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SelectedStream {
    pub stream_name: String,
    #[serde(default)]
    pub partition_regex: String,
    pub normalization: bool,
    #[serde(default)]
    pub filter: Option<String>,
    #[serde(default)]
    pub disabled: Option<bool>,
    #[serde(default)]
    pub append_mode: Option<bool>,
    #[serde(default)]
    pub selected_columns: Option<SelectedColumns>,
}

/// Key = namespace string, value = list of selected streams.
pub type SelectedStreamsByNamespace = HashMap<String, Vec<SelectedStream>>;

/// The full discover result / streams config structure.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StreamsDataStructure {
    pub selected_streams: SelectedStreamsByNamespace,
    pub streams: Vec<StreamData>,
}

/// Alias kept for compatibility.
pub type DiscoverResult = StreamsDataStructure;

/// Request body for the discover (streams) endpoint.
#[derive(Debug, Clone, Serialize)]
pub struct DiscoverRequest {
    pub name: String,
    #[serde(rename = "type")]
    pub source_type: String,
    pub job_name: String,
    pub job_id: i64,
    pub version: String,
    pub config: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_discover_threads: Option<u32>,
}

// ---------------------------------------------------------------------------
// Job Types
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum JobType {
    Sync,
    Clear,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdvancedSettings {
    #[serde(default)]
    pub max_discover_threads: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JobConnector {
    #[serde(default)]
    pub id: Option<i64>,
    pub name: String,
    #[serde(rename = "type")]
    pub connector_type: String,
    pub version: String,
    pub config: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Job {
    pub id: i64,
    pub name: String,
    pub source: JobConnector,
    pub destination: JobConnector,
    pub streams_config: String,
    pub frequency: String,
    #[serde(default)]
    pub last_run_type: Option<JobType>,
    #[serde(default)]
    pub last_run_state: String,
    #[serde(default)]
    pub last_run_time: String,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub updated_at: String,
    #[serde(default)]
    pub created_by: String,
    #[serde(default)]
    pub updated_by: String,
    pub activate: bool,
    #[serde(default)]
    pub advanced_settings: Option<AdvancedSettings>,
}

#[derive(Debug, Clone, Serialize)]
pub struct JobBase {
    pub name: String,
    pub source: JobConnector,
    pub destination: JobConnector,
    pub frequency: String,
    pub streams_config: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub difference_streams: Option<String>,
    #[serde(default)]
    pub activate: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub advanced_settings: Option<AdvancedSettings>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct JobTask {
    pub runtime: String,
    pub start_time: String,
    pub status: String,
    pub file_path: String,
    pub job_type: JobType,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TaskLogsDirection {
    #[serde(rename = "older")]
    Older,
    #[serde(rename = "newer")]
    Newer,
}

#[derive(Debug, Clone, Serialize)]
pub struct TaskLogsPaginationParams {
    pub cursor: i64,
    pub limit: u32,
    pub direction: TaskLogsDirection,
}

impl Default for TaskLogsPaginationParams {
    fn default() -> Self {
        Self {
            cursor: -1,
            limit: 1000,
            direction: TaskLogsDirection::Older,
        }
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct TaskLogsResponse {
    pub logs: Vec<LogEntry>,
    pub older_cursor: i64,
    pub newer_cursor: i64,
    pub has_more_older: bool,
    pub has_more_newer: bool,
}

#[derive(Debug, Clone, Serialize)]
pub struct TaskLogsRequest {
    pub file_path: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct ActivateRequest {
    pub activate: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ActivateResponse {
    pub activate: bool,
}

#[derive(Debug, Clone, Serialize)]
pub struct StreamDifferenceRequest {
    pub updated_streams_config: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct StreamDifferenceResponse {
    pub difference_streams: StreamsDataStructure,
}

// ---------------------------------------------------------------------------
// Settings Types
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct SystemSettings {
    pub id: i64,
    pub project_id: String,
    pub webhook_alert_url: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct UpdateSystemSettingsRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<i64>,
    pub project_id: String,
    pub webhook_alert_url: String,
}

// ---------------------------------------------------------------------------
// Platform / Release Types
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct ReleaseMetadata {
    #[serde(default)]
    pub title: Option<String>,
    #[serde(default)]
    pub version: Option<String>,
    pub description: String,
    #[serde(default)]
    pub tags: Vec<String>,
    pub date: String,
    pub link: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ReleaseTypeData {
    #[serde(default)]
    pub current_version: Option<String>,
    pub releases: Vec<ReleaseMetadata>,
}

#[derive(Debug, Clone, Deserialize, Default)]
pub struct ReleasesResponse {
    #[serde(default)]
    pub features: Option<ReleaseTypeData>,
    #[serde(default)]
    pub olake_ui_worker: Option<ReleaseTypeData>,
    #[serde(default)]
    pub olake_helm: Option<ReleaseTypeData>,
    #[serde(default)]
    pub olake: Option<ReleaseTypeData>,
}

// ---------------------------------------------------------------------------
// Check Unique
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize)]
pub struct CheckUniqueRequest {
    pub name: String,
    pub entity_type: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CheckUniqueResponse {
    pub unique: bool,
}

// ---------------------------------------------------------------------------
// Delete response (name only)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct DeleteResponse {
    pub name: String,
}

// ---------------------------------------------------------------------------
// Message response (generic message string)
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct MessageResponse {
    pub message: String,
}

// ---------------------------------------------------------------------------
// Cancel response
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct CancelResponse {
    pub message: String,
}

// ---------------------------------------------------------------------------
// ClearDestinationStatus
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Deserialize)]
pub struct ClearDestinationStatus {
    pub running: bool,
}
