# OLake TUI vs BFF — Feature Comparison & Test Checklist

> Created: 2026-03-19
> BFF reference: `datazip-inc/olake-ui/server/`

## Architecture Differences

| Area | BFF (olake-ui server) | olake-tui |
|------|----------------------|-----------|
| DB access | Beego ORM | database/sql (lib/pq) |
| Temporal | temporal wrapper package | go.temporal.io/sdk direct |
| Encryption | AES-256-GCM (utils/encryption.go) | ✅ Same implementation |
| Auth | bcrypt + session cookie | ✅ bcrypt (direct DB query) |
| Table naming | `olake-{runMode}-{entity}` | ✅ Same |
| Soft delete | Beego ORM (deleted_at) | ✅ Manual implementation |

---

## Feature Comparison

### ✅ Fully Implemented

| BFF Feature | TUI Method | Notes |
|-------------|-----------|-------|
| Login | `Login()` | Same bcrypt verification |
| ListSources | `ListSources()` | Includes job_count |
| GetSource | `GetSource()` | — |
| CreateSource | `CreateSource()` | With encryption |
| UpdateSource | `UpdateSource()` | — |
| DeleteSource | `DeleteSource()` | Soft-delete + job count check |
| ListDestinations | `ListDestinations()` | Includes job_count |
| GetDestination | `GetDestination()` | — |
| CreateDestination | `CreateDestination()` | — |
| UpdateDestination | `UpdateDestination()` | — |
| DeleteDestination | `DeleteDestination()` | Soft-delete + job count check |
| ListJobs | `ListJobs()` | Includes source/dest JOIN |
| GetJob | `GetJob()` | — |
| CreateJob | `CreateJob()` | ✅ Includes Temporal schedule creation |
| DeleteJob | `DeleteJob()` | Includes Temporal schedule deletion |
| TriggerSync | `TriggerSync()` | schedule.Trigger() |
| CancelJob | `CancelJob()` | Workflow cancel |
| ActivateJob | `ActivateJob()` | DB + schedule pause/unpause |
| UpdateJobMeta | `UpdateJobMeta()` | Name + frequency |
| GetJobTasks | `ListJobTasks()` | Temporal workflow history |
| GetTaskLogs | `GetTaskLogs()` | Disk log file reader |
| TestSourceConnection | `TestSource()` | Temporal check workflow |
| TestDestConnection | `TestDestination()` | Temporal check workflow |
| DiscoverStreams | `DiscoverStreams()` | Temporal discover workflow |
| ClearDestination | `ClearDestination()` | 2-step confirm + schedule swap |
| GetSettings | `GetSettings()` | project-settings table |
| UpdateSettings | `UpdateSettings()` | UPSERT |
| ValidateSchema | `ValidateSchema()` | Table existence check |

### ⚠️ Partial Implementation (behavioral differences)

| BFF Feature | TUI Status | Difference |
|-------------|-----------|------------|
| CreateJob | ✅ Implemented | BFF: upserts source/dest (reuses existing) → TUI: passes sourceID/destID directly |
| DeleteSource/Dest | ✅ Implemented | BFF: cascade cancels related jobs → TUI: rejects deletion if jobs exist (more conservative) |

### ❌ Not Implemented (non-blocking — core features work without these)

| BFF Feature | Description | Priority | Notes |
|-------------|-------------|----------|-------|
| `GetSourceVersions` | Source connector version list (Docker image tags) | 🟡 | BFF calls GitHub API |
| `GetSourceSpec` | Connector spec JSON (auto-generated form fields) | 🟡 | TUI uses hardcoded forms instead |
| `GetDestinationVersions` | Destination connector version list | 🟡 | Same as above |
| `GetDestinationSpec` | Destination connector spec JSON | 🟡 | Same as above |
| `GetAllReleasesResponse` | GitHub releases list (update notifications) | 🟢 | Updates modal placeholder |
| `DownloadTaskLogs` | Log tar.gz download | 🟢 | TUI has direct filesystem access |
| Telemetry tracking | Job/source creation telemetry | 🟢 | OLake internal analytics |
| `CheckClearDestCompatibility` | Source version clear-dest compatibility check | 🟡 | Version string comparison |

### ✅ Added After Gap Analysis

| BFF Feature | TUI Implementation |
|-------------|-------------------|
| `UpdateJob` (full) | `UpdateJobFull()` — all fields + clear-dest blocking + sync cancellation |
| Job name uniqueness | `IsNameUnique()` + duplicate rejection in CreateJob/Source/Dest |
| `GetClearDestinationStatus` | `GetClearDestStatus()` |
| `RecoverFromClearDestination` | `RecoverFromClearDest()` + UI button |
| ListJobs real-time status | `fetchJobLastRun()` — live Temporal query (5s timeout) |

---

## E2E Test Checklist

Items to verify against a real OLake environment:

### Authentication
- [ ] Successful login with valid credentials
- [ ] Failed login with invalid credentials + error message
- [ ] Client-side empty username/password validation

### Sources
- [ ] List sources (job count accuracy)
- [ ] Create PostgreSQL source
- [ ] Edit source name/config
- [ ] Delete source (no associated jobs)
- [ ] Delete source rejected (has associated jobs)
- [ ] Connection test (Temporal check workflow)
- [ ] Encrypted config decryptable by BFF

### Destinations
- [ ] List destinations
- [ ] Create Iceberg/Parquet destination
- [ ] Edit destination
- [ ] Delete destination
- [ ] Connection test

### Jobs
- [ ] List jobs (source/dest name mapping)
- [ ] Create job → verify Temporal schedule (`tctl schedule list`)
- [ ] Trigger sync (manual)
- [ ] Cancel running sync (workflow cancel)
- [ ] Pause/resume job (schedule pause/unpause)
- [ ] Delete job → verify Temporal schedule removed
- [ ] Edit job name/frequency
- [ ] Task history listing
- [ ] Task log viewer (pagination: older/newer)

### Streams
- [ ] Discover streams from source
- [ ] Select/deselect streams
- [ ] Per-stream sync mode configuration
- [ ] Cursor field configuration

### Settings
- [ ] Retrieve webhook URL
- [ ] Save webhook URL

### Clear Destination
- [ ] Execute clear destination (2-step confirmation)
- [ ] Verify Temporal schedule restored to sync after completion

### Cross-Compatibility
- [ ] Source created by TUI visible/editable in BFF web UI
- [ ] Job created by BFF manageable in TUI (list/sync/cancel)
- [ ] Bidirectional config decryption with same encryption key
- [ ] Soft-deleted items hidden in both TUI and BFF

### Edge Cases
- [ ] TUI runs without Temporal connection → DB features work
- [ ] Invalid DB URL → clear error message
- [ ] Invalid runMode → rejected
- [ ] Narrow terminal (80x24) → no layout breakage
- [ ] Very long source/job names → truncated properly

---

## Implementation Roadmap

### ✅ Phase 1 — Complete (core BFF compatibility)
1. ~~`UpdateJob` full implementation~~ → `UpdateJobFull` ✅
2. ~~Job name uniqueness~~ → `IsNameUnique` ✅
3. ~~ListJobs real-time status~~ → `fetchJobLastRun` ✅
4. ~~Clear-dest status/recovery~~ → `GetClearDestStatus` + `RecoverFromClearDest` ✅

### Phase 2 (optional — UX improvements)
5. `GetSourceSpec` / `GetDestinationSpec` — spec-based dynamic forms
6. `GetSourceVersions` / `GetDestinationVersions` — version selector
7. `CheckClearDestCompatibility` — source version compatibility check

### Phase 3 (low priority)
8. `GetAllReleasesResponse` — real release data in Updates modal
9. `DownloadTaskLogs` — log tar.gz download
10. Telemetry (OLake internal analytics)
