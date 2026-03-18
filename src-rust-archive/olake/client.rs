//! HTTP client for the OLake BFF server.
//!
//! Base URL is read from the `OLAKE_API_URL` environment variable,
//! defaulting to `http://localhost:8000`.
//!
//! Session is maintained via a cookie jar (the BFF uses session cookies).
//! An `Authorization: Bearer authenticated` header is also sent on every
//! request after a successful login, mirroring the browser frontend's
//! behaviour.

use std::sync::Arc;

use reqwest::{
    cookie::Jar,
    header::{HeaderMap, HeaderValue, AUTHORIZATION, CONTENT_TYPE},
    Client, StatusCode,
};
use thiserror::Error;
use tokio::sync::RwLock;

use crate::olake::types::*;

const DEFAULT_BASE_URL: &str = "http://localhost:8000";
const PROJECT_ID: &str = "123";

// ---------------------------------------------------------------------------
// Error type
// ---------------------------------------------------------------------------

#[derive(Debug, Error)]
pub enum OlakeError {
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),

    #[error("API error ({status}): {message}")]
    Api { status: StatusCode, message: String },

    #[error("Unexpected response shape: {0}")]
    Deserialize(String),

    #[error("Not authenticated — call login() first")]
    Unauthenticated,

    #[error("Forbidden")]
    Forbidden,

    #[error("Server error: {0}")]
    ServerError(String),

    #[error("{0}")]
    Other(String),
}

// ---------------------------------------------------------------------------
// Auth state (shared between callers)
// ---------------------------------------------------------------------------

#[derive(Debug, Default, Clone)]
struct AuthState {
    /// Mirrors localStorage["token"]. `Some("authenticated")` when logged in.
    token: Option<String>,
    username: Option<String>,
}

// ---------------------------------------------------------------------------
// OlakeClient
// ---------------------------------------------------------------------------

#[derive(Clone)]
pub struct OlakeClient {
    base_url: String,
    client: Client,
    auth: Arc<RwLock<AuthState>>,
}

impl OlakeClient {
    /// Create a new client. Base URL is taken from `OLAKE_API_URL` env var
    /// (default `http://localhost:8000`).
    pub fn new() -> Result<Self, OlakeError> {
        let base_url = std::env::var("OLAKE_API_URL")
            .unwrap_or_else(|_| DEFAULT_BASE_URL.to_owned());

        let jar = Arc::new(Jar::default());
        let client = Client::builder()
            .cookie_provider(jar)
            .timeout(std::time::Duration::from_secs(10))
            .build()?;

        Ok(Self {
            base_url,
            client,
            auth: Arc::new(RwLock::new(AuthState::default())),
        })
    }

    // -----------------------------------------------------------------------
    // Internal helpers
    // -----------------------------------------------------------------------

    fn project_url(&self, path: &str) -> String {
        format!("{}/api/v1/project/{}/{}", self.base_url, PROJECT_ID, path)
    }

    fn platform_url(&self, path: &str) -> String {
        format!("{}/api/v1/platform/{}", self.base_url, path)
    }

    fn root_url(&self, path: &str) -> String {
        format!("{}/{}", self.base_url, path)
    }

    async fn auth_headers(&self) -> HeaderMap {
        let mut headers = HeaderMap::new();
        headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));

        let auth = self.auth.read().await;
        if let Some(token) = &auth.token {
            if let Ok(val) = HeaderValue::from_str(&format!("Bearer {}", token)) {
                headers.insert(AUTHORIZATION, val);
            }
        }
        headers
    }

    /// Unwrap the standard `{ success, message, data }` envelope.
    async fn unwrap_response<T: serde::de::DeserializeOwned>(
        &self,
        resp: reqwest::Response,
    ) -> Result<T, OlakeError> {
        let status = resp.status();

        match status {
            StatusCode::UNAUTHORIZED => {
                self.clear_auth().await;
                return Err(OlakeError::Unauthenticated);
            }
            StatusCode::FORBIDDEN => return Err(OlakeError::Forbidden),
            s if s.is_server_error() => {
                let msg = resp
                    .text()
                    .await
                    .unwrap_or_else(|_| "Server error".to_owned());
                return Err(OlakeError::ServerError(msg));
            }
            _ => {}
        }

        let envelope: ApiResponse<T> = resp
            .json()
            .await
            .map_err(|e| OlakeError::Deserialize(e.to_string()))?;

        if !envelope.success {
            return Err(OlakeError::Api {
                status,
                message: envelope.message,
            });
        }

        Ok(envelope.data)
    }

    async fn clear_auth(&self) {
        let mut auth = self.auth.write().await;
        auth.token = None;
        auth.username = None;
    }

    // -----------------------------------------------------------------------
    // Auth domain
    // -----------------------------------------------------------------------

    /// POST /login
    pub async fn login(
        &self,
        username: impl Into<String>,
        password: impl Into<String>,
    ) -> Result<LoginResponse, OlakeError> {
        let body = LoginRequest {
            username: username.into(),
            password: password.into(),
        };

        let resp = self
            .client
            .post(self.root_url("login"))
            .headers(self.auth_headers().await)
            .json(&body)
            .send()
            .await?;

        let data: LoginResponse = self.unwrap_response(resp).await?;

        // Mirror frontend: store token + username
        {
            let mut auth = self.auth.write().await;
            auth.token = Some("authenticated".to_owned());
            auth.username = Some(data.username.clone());
        }

        Ok(data)
    }

    /// POST /signup
    pub async fn signup(
        &self,
        username: impl Into<String>,
        password: impl Into<String>,
        email: impl Into<String>,
    ) -> Result<SignupResponse, OlakeError> {
        let body = SignupRequest {
            email: email.into(),
            username: username.into(),
            password: password.into(),
        };

        let resp = self
            .client
            .post(self.root_url("signup"))
            .headers(self.auth_headers().await)
            .json(&body)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /auth — verify the current session is still valid.
    pub async fn check_auth(&self) -> Result<AuthCheckResponse, OlakeError> {
        let resp = self
            .client
            .get(self.root_url("auth"))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /telemetry-id
    pub async fn get_telemetry_id(&self) -> Result<TelemetryIdResponse, OlakeError> {
        let resp = self
            .client
            .get(self.root_url("telemetry-id"))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Sources domain
    // -----------------------------------------------------------------------

    /// GET /api/v1/project/:id/sources
    pub async fn sources_list(&self) -> Result<Vec<Entity>, OlakeError> {
        let resp = self
            .client
            .get(self.project_url("sources"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/sources/:source_id
    pub async fn sources_get(&self, id: i64) -> Result<Entity, OlakeError> {
        let resp = self
            .client
            .get(self.project_url(&format!("sources/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/sources
    pub async fn sources_create(&self, entity: EntityBase) -> Result<EntityBase, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("sources"))
            .headers(self.auth_headers().await)
            .json(&entity)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// PUT /api/v1/project/:id/sources/:source_id
    pub async fn sources_update(&self, id: i64, entity: EntityBase) -> Result<Entity, OlakeError> {
        let resp = self
            .client
            .put(self.project_url(&format!("sources/{}", id)))
            .headers(self.auth_headers().await)
            .json(&entity)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// DELETE /api/v1/project/:id/sources/:source_id
    pub async fn sources_delete(&self, id: i64) -> Result<DeleteResponse, OlakeError> {
        let resp = self
            .client
            .delete(self.project_url(&format!("sources/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/sources/test
    pub async fn sources_test_connection(
        &self,
        config: EntityTestRequest,
    ) -> Result<EntityTestResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("sources/test"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(600))
            .json(&config)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/sources/streams (discover catalog)
    pub async fn sources_discover(
        &self,
        source_id: i64,
        config: DiscoverRequest,
    ) -> Result<DiscoverResult, OlakeError> {
        let _ = source_id; // source_id is embedded in config.job_id
        let resp = self
            .client
            .post(self.project_url("sources/streams"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(600))
            .json(&config)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/sources/spec
    pub async fn sources_get_spec(&self, driver: SpecRequest) -> Result<SpecResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("sources/spec"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(300))
            .json(&driver)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/sources/versions/?type=:driver
    pub async fn sources_get_versions(&self, driver: &str) -> Result<VersionsResponse, OlakeError> {
        let url = format!("{}/api/v1/project/{}/sources/versions/", self.base_url, PROJECT_ID);
        let resp = self
            .client
            .get(&url)
            .headers(self.auth_headers().await)
            .query(&[("type", driver)])
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Destinations domain
    // -----------------------------------------------------------------------

    /// GET /api/v1/project/:id/destinations
    pub async fn destinations_list(&self) -> Result<Vec<Entity>, OlakeError> {
        let resp = self
            .client
            .get(self.project_url("destinations"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/destinations/:dest_id
    pub async fn destinations_get(&self, id: i64) -> Result<Entity, OlakeError> {
        let resp = self
            .client
            .get(self.project_url(&format!("destinations/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/destinations
    pub async fn destinations_create(&self, entity: EntityBase) -> Result<EntityBase, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("destinations"))
            .headers(self.auth_headers().await)
            .json(&entity)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// PUT /api/v1/project/:id/destinations/:dest_id
    pub async fn destinations_update(
        &self,
        id: i64,
        entity: EntityBase,
    ) -> Result<EntityBase, OlakeError> {
        let resp = self
            .client
            .put(self.project_url(&format!("destinations/{}", id)))
            .headers(self.auth_headers().await)
            .json(&entity)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// DELETE /api/v1/project/:id/destinations/:dest_id
    pub async fn destinations_delete(&self, id: i64) -> Result<DeleteResponse, OlakeError> {
        let resp = self
            .client
            .delete(self.project_url(&format!("destinations/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/destinations/test
    pub async fn destinations_test_connection(
        &self,
        config: EntityTestRequest,
    ) -> Result<EntityTestResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("destinations/test"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(600))
            .json(&config)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/destinations/spec
    pub async fn destinations_get_spec(
        &self,
        dtype: SpecRequest,
    ) -> Result<SpecResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("destinations/spec"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(300))
            .json(&dtype)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/destinations/versions/?type=:dtype
    pub async fn destinations_get_versions(
        &self,
        dtype: &str,
    ) -> Result<VersionsResponse, OlakeError> {
        let url = format!(
            "{}/api/v1/project/{}/destinations/versions/",
            self.base_url, PROJECT_ID
        );
        let resp = self
            .client
            .get(&url)
            .headers(self.auth_headers().await)
            .query(&[("type", dtype)])
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Jobs domain
    // -----------------------------------------------------------------------

    /// GET /api/v1/project/:id/jobs
    pub async fn jobs_list(&self) -> Result<Vec<Job>, OlakeError> {
        let resp = self
            .client
            .get(self.project_url("jobs"))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/jobs/:job_id
    pub async fn jobs_get(&self, id: i64) -> Result<Job, OlakeError> {
        let resp = self
            .client
            .get(self.project_url(&format!("jobs/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/jobs
    pub async fn jobs_create(&self, job: JobBase) -> Result<Job, OlakeError> {
        let resp = self
            .client
            .post(self.project_url("jobs"))
            .headers(self.auth_headers().await)
            .json(&job)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// PUT /api/v1/project/:id/jobs/:job_id
    pub async fn jobs_update(&self, id: i64, job: JobBase) -> Result<Job, OlakeError> {
        let resp = self
            .client
            .put(self.project_url(&format!("jobs/{}", id)))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(30))
            .json(&job)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// DELETE /api/v1/project/:id/jobs/:job_id
    pub async fn jobs_delete(&self, id: i64) -> Result<DeleteResponse, OlakeError> {
        let resp = self
            .client
            .delete(self.project_url(&format!("jobs/{}", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/jobs/:job_id/sync
    pub async fn jobs_sync(&self, id: i64) -> Result<serde_json::Value, OlakeError> {
        let resp = self
            .client
            .post(self.project_url(&format!("jobs/{}/sync", id)))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(0))
            .json(&serde_json::json!({}))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/jobs/:job_id/cancel
    pub async fn jobs_cancel(
        &self,
        id: i64,
        _task_id: Option<&str>,
    ) -> Result<CancelResponse, OlakeError> {
        let resp = self
            .client
            .get(self.project_url(&format!("jobs/{}/cancel", id)))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/jobs/:job_id/activate
    pub async fn jobs_activate(
        &self,
        id: i64,
        activate: bool,
    ) -> Result<ActivateResponse, OlakeError> {
        let body = ActivateRequest { activate };
        let resp = self
            .client
            .post(self.project_url(&format!("jobs/{}/activate", id)))
            .headers(self.auth_headers().await)
            .json(&body)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/jobs/:job_id/tasks
    pub async fn jobs_get_tasks(&self, id: i64) -> Result<Vec<JobTask>, OlakeError> {
        let resp = self
            .client
            .get(self.project_url(&format!("jobs/{}/tasks", id)))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(0))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/jobs/:job_id/tasks/:task_id/logs
    pub async fn jobs_get_logs(
        &self,
        job_id: i64,
        task_id: &str,
        params: TaskLogsPaginationParams,
        file_path: &str,
    ) -> Result<TaskLogsResponse, OlakeError> {
        let body = TaskLogsRequest {
            file_path: file_path.to_owned(),
        };
        let url = self.project_url(&format!("jobs/{}/tasks/{}/logs", job_id, task_id));
        let resp = self
            .client
            .post(&url)
            .headers(self.auth_headers().await)
            .query(&[
                ("cursor", params.cursor.to_string()),
                ("limit", params.limit.to_string()),
                (
                    "direction",
                    match params.direction {
                        TaskLogsDirection::Older => "older".to_owned(),
                        TaskLogsDirection::Newer => "newer".to_owned(),
                    },
                ),
            ])
            .json(&body)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// GET /api/v1/project/:id/jobs/:job_id/logs/download?file_path=...
    pub async fn jobs_download_logs(
        &self,
        job_id: i64,
        task_id: &str,
    ) -> Result<bytes::Bytes, OlakeError> {
        let _ = task_id; // task_id is provided via file_path query param in real usage
        let url = self.project_url(&format!("jobs/{}/logs/download", job_id));
        let resp = self
            .client
            .get(&url)
            .headers(self.auth_headers().await)
            .send()
            .await?;

        let status = resp.status();
        if status.is_server_error() {
            let msg = resp.text().await.unwrap_or_else(|_| "Server error".to_owned());
            return Err(OlakeError::ServerError(msg));
        }
        if status == StatusCode::UNAUTHORIZED {
            self.clear_auth().await;
            return Err(OlakeError::Unauthenticated);
        }

        let bytes = resp.bytes().await?;
        Ok(bytes)
    }

    /// POST /api/v1/project/:id/jobs/:job_id/clear-destination
    pub async fn jobs_clear_destination(&self, id: i64) -> Result<MessageResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url(&format!("jobs/{}/clear-destination", id)))
            .headers(self.auth_headers().await)
            .json(&serde_json::json!({}))
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// POST /api/v1/project/:id/jobs/:job_id/stream-difference
    pub async fn jobs_stream_difference(
        &self,
        id: i64,
        config: StreamDifferenceRequest,
    ) -> Result<StreamDifferenceResponse, OlakeError> {
        let resp = self
            .client
            .post(self.project_url(&format!("jobs/{}/stream-difference", id)))
            .headers(self.auth_headers().await)
            .timeout(std::time::Duration::from_secs(30))
            .json(&config)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Settings domain
    // -----------------------------------------------------------------------

    /// GET /api/v1/project/:id/settings
    pub async fn settings_get(&self) -> Result<SystemSettings, OlakeError> {
        let resp = self
            .client
            .get(self.project_url("settings"))
            .headers(self.auth_headers().await)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    /// PUT /api/v1/project/:id/settings
    pub async fn settings_update(
        &self,
        settings: UpdateSystemSettingsRequest,
    ) -> Result<serde_json::Value, OlakeError> {
        let resp = self
            .client
            .put(self.project_url("settings"))
            .headers(self.auth_headers().await)
            .json(&settings)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Platform domain
    // -----------------------------------------------------------------------

    /// GET /api/v1/platform/releases?limit=:limit
    pub async fn platform_releases(
        &self,
        limit: Option<u32>,
    ) -> Result<ReleasesResponse, OlakeError> {
        let url = self.platform_url("releases");
        let mut req = self
            .client
            .get(&url)
            .headers(self.auth_headers().await);

        if let Some(l) = limit {
            req = req.query(&[("limit", l.to_string())]);
        }

        let resp = req.send().await?;
        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Utility
    // -----------------------------------------------------------------------

    /// POST /api/v1/project/:id/check-unique
    pub async fn check_unique(
        &self,
        name: &str,
        entity_type: &str,
    ) -> Result<CheckUniqueResponse, OlakeError> {
        let body = CheckUniqueRequest {
            name: name.to_owned(),
            entity_type: entity_type.to_owned(),
        };
        let resp = self
            .client
            .post(self.project_url("check-unique"))
            .headers(self.auth_headers().await)
            .json(&body)
            .send()
            .await?;

        self.unwrap_response(resp).await
    }

    // -----------------------------------------------------------------------
    // Auth helpers
    // -----------------------------------------------------------------------

    /// Returns the currently authenticated username, if any.
    pub async fn username(&self) -> Option<String> {
        self.auth.read().await.username.clone()
    }

    /// Returns `true` if a token is stored (i.e., login succeeded).
    pub async fn is_authenticated(&self) -> bool {
        self.auth.read().await.token.is_some()
    }

    /// Clear stored auth state (logout).
    pub async fn logout(&self) {
        self.clear_auth().await;
    }
}
