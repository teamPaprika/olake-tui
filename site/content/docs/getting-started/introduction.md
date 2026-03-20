---
title: "Introduction"
weight: 1
---

# What is olake-tui?

**olake-tui** is a terminal user interface for [OLake](https://olake.io/docs/) — the open-source data pipeline platform. It lets you manage sources, destinations, and sync jobs entirely from the command line, without a web browser or BFF server.

olake-tui connects **directly** to the OLake PostgreSQL database and Temporal cluster, bypassing the HTTP layer. This means fewer moving parts, faster feedback, and full control from any terminal.

```
┌──────────────────────────────────────────────────────────────────┐
│  ⬡ OLake  logged in as admin                                     │
│                                                                   │
│  ╭──────────╮  ╭──────────────╮  ╭──────╮  ╭────────────╮  ╭───╮│
│  │ Sources: 3│  │ Destinations:2│  │Jobs:5│  │Active Jobs:3│  │⟳:1││
│  ╰──────────╯  ╰──────────────╯  ╰──────╯  ╰────────────╯  ╰───╯│
│                                                                   │
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings            │
│                                                                   │
│  ✓  1  nightly-postgres-sync   pg-prod    s3-bucket  completed  ● │
│  ⟳  2  hourly-mysql-export     mysql-dev  iceberg    running    ● │
│  ✗  3  daily-mongo-backup      mongo      s3-archive failed     ● │
│  ·  4  weekly-report           pg-prod    parquet    —          ○ │
│                                                                   │
│  n:new  Enter:detail  S:settings  s:sync  c:cancel  l:logs       │
└──────────────────────────────────────────────────────────────────┘
```

## Why a TUI?

The OLake web UI works well for teams with browser access and a running BFF server. But many real-world environments don't fit that model:

### Air-gapped and restricted networks

Production environments often have no outbound internet or browser access. olake-tui runs over SSH — if you can reach the database, you can manage your pipelines.

### SSH-first workflows

Jump boxes, bastion hosts, remote servers. When your only interface is a terminal, a TUI is the natural fit. No port forwarding for web UIs, no VPN tunnels for dashboards.

### CI/CD and automation

olake-tui supports non-interactive flags (`--migrate`, `--migrate-only`) that make it usable in init containers, setup scripts, and CI pipelines. Bootstrap your database schema without spinning up a web stack.

### Keyboard-driven speed

Navigate jobs, trigger syncs, inspect logs — all without touching a mouse. Power users move faster in a terminal than clicking through web pages.

### Fewer dependencies

The standard OLake deployment requires a BFF server, signup container, and web frontend. olake-tui replaces all three. You need only:

- **PostgreSQL** — stores sources, destinations, jobs, and settings
- **Temporal** — orchestrates sync workflows
- **OLake Worker** — executes the actual data movement

No Node.js server. No React app. No nginx.

## What can it do?

olake-tui provides full CRUD operations for the OLake platform:

| Feature | Description |
|---------|-------------|
| **Sources** | Create, edit, delete, and test database connections (MongoDB, PostgreSQL, MySQL) |
| **Destinations** | Manage output targets (S3, Apache Iceberg, local Parquet) |
| **Jobs** | Create jobs with stream selection, configure sync modes (full refresh / CDC), trigger syncs |
| **Logs** | Paginated log viewer for completed and running tasks |
| **Settings** | View and update OLake configuration |
| **Auth** | Login with the same credentials as the web UI |

The TUI reads and writes the **same database schema** as the BFF server. You can switch between the web UI and TUI freely — they are fully compatible.

## Architecture overview

```
┌─────────────┐
│  olake-tui   │  ← You are here
└──────┬───────┘
       │ Direct SQL + Temporal gRPC
       ▼
┌──────────────┐     ┌──────────────┐
│  PostgreSQL   │     │   Temporal    │
│  (shared DB)  │     │  (workflows)  │
└──────────────┘     └──────┬───────┘
                            │
                     ┌──────▼───────┐
                     │  OLake Worker │
                     │  (sync/check) │
                     └──────────────┘
```

olake-tui talks to PostgreSQL for all CRUD operations and to Temporal for workflow state (job status, sync triggers, schedule management). The OLake Worker picks up workflows from Temporal and executes the actual data movement.

## Next steps

- [Quick Start]({{< relref "quick-start" >}}) — Get running in 5 minutes
- [Installation]({{< relref "installation" >}}) — Build from source or use Docker
- [OLake Documentation](https://olake.io/docs/) — Learn about the OLake platform itself
