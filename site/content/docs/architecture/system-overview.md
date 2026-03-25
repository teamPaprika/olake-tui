---
title: "System Overview"
weight: 1
---


olake-tui is a terminal-based management interface for [OLake](https://olake.io) that connects **directly** to PostgreSQL and Temporal — no HTTP API layer in between.

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│                  olake-tui                       │
│  ┌───────────┐  ┌───────────┐  ┌─────────────┐ │
│  │  Bubble Tea│  │  Service   │  │  UI Screens │ │
│  │  Runtime   │──│  Manager   │──│  (13 views) │ │
│  └───────────┘  └─────┬─────┘  └─────────────┘ │
│                       │                          │
└───────────────────────┼──────────────────────────┘
                        │
            ┌───────────┴───────────┐
            │                       │
     ┌──────▼──────┐        ┌──────▼──────┐
     │  PostgreSQL  │        │   Temporal   │
     │  (metadata)  │        │  (workflows) │
     └─────────────┘        └─────────────┘
```

## No HTTP Layer

Unlike the traditional OLake web UI which routes through a BFF (Backend-For-Frontend) HTTP server, olake-tui eliminates the middleware entirely:

| Aspect | Web UI (BFF) | olake-tui |
|--------|-------------|-----------|
| Data access | Browser → BFF API → PostgreSQL | TUI → PostgreSQL |
| Workflow control | Browser → BFF API → Temporal | TUI → Temporal |
| Authentication | JWT over HTTP | Local DB session |
| Deployment | 3+ services | Single binary |
| Latency | Two network hops | Direct connection |

This design means fewer moving parts, simpler deployment, and faster operations. The TUI binary is the only process you need to run — it talks directly to the same PostgreSQL and Temporal instances that the web UI uses.

## Component Flow

A typical user session follows this path:

```
Login
  │
  ▼
Dashboard ──────────────────────────────────┐
  │                                          │
  ├── Sources ── Create/Edit/Delete          │
  │                                          │
  ├── Destinations ── Create/Edit/Delete     │
  │                                          │
  ├── Jobs ── Create (Wizard) ───────────┐   │
  │       ├── Detail View                │   │
  │       ├── Settings                   │   │
  │       ├── Stream Selection           │   │
  │       └── Logs (real-time)           │   │
  │                                      │   │
  ├── Sync (trigger Temporal workflow)   │   │
  │                                      │   │
  └── Settings (project-level config)    │   │
                                         │   │
  Temporal ◄─── Schedule/Run ────────────┘   │
     │                                       │
     └── Workflow logs streamed back ────────┘
```

### Step-by-step

1. **Login** — Authenticate against the `user` table in PostgreSQL. A session record is created.
2. **CRUD** — Create and manage sources, destinations, and jobs. All metadata is stored in PostgreSQL using the same schema as the BFF.
3. **Sync** — Trigger a Temporal workflow to run a data sync job. The TUI creates or updates Temporal schedules directly.
4. **Logs** — Stream workflow execution logs in real-time from Temporal, displayed in the TUI's log viewer.

## Data Compatibility

Because olake-tui uses the **identical database schema** and **identical Temporal naming conventions** as the BFF-based web UI, you can switch between them freely. Sources, destinations, and jobs created in the TUI appear in the web UI and vice versa.

See [BFF Compatibility](../bff-compatibility/) for full details.

## External Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| PostgreSQL | 13+ | Metadata storage (sources, destinations, jobs, users) |
| Temporal | 1.22+ | Workflow orchestration, job scheduling, log retrieval |

Both must be running and accessible before starting olake-tui. Connection details are provided via CLI flags or environment variables.

## Further Reading

- [Package Structure](../package-structure/) — How the codebase is organized
- [Database Schema](../database-schema/) — Table definitions and naming conventions
- [OLake Architecture](https://olake.io/blog/architecture) — Overall OLake platform architecture
