---
title: "Introduction"
weight: 1
---

# What is olake-tui?

**olake-tui** is a terminal user interface for [OLake](https://olake.io/docs/) — the open-source data pipeline platform. It lets you manage sources, destinations, and sync jobs entirely from the command line, without a web browser or BFF server.

If you've used OLake's web UI before, think of olake-tui as its keyboard-driven twin: same database, same workflows, zero browser required.

## What is OLake?

[OLake](https://olake.io/) is an open-source platform for building data pipelines. It connects to source databases (PostgreSQL, MySQL, MongoDB, Oracle, MSSQL, DB2, Kafka, S3), extracts data, and writes it to destinations (Apache Iceberg, Amazon S3 as Parquet). OLake supports both full-refresh and CDC (Change Data Capture) sync modes.

For full OLake documentation, see [olake.io/docs](https://olake.io/docs/).

## What is olake-tui?

olake-tui is a standalone Go binary (~10MB) that replaces the entire OLake web stack — the React frontend, the BFF (Backend-for-Frontend) Node.js server, and the signup-init container. It connects **directly** to PostgreSQL and Temporal via SQL and gRPC, with no HTTP layer in between.

Here's what it looks like in your terminal:

```
┌──────────────────────────────────────────────────────────────────┐
│  ⬡ OLake  logged in as admin                             v0.2.0 │
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

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (the Go TUI framework), it provides a rich interactive experience — forms, lists, spinners, toast notifications — all rendered in your terminal.

## Architecture

olake-tui eliminates the HTTP layer entirely. The web UI talks to a BFF server, which talks to PostgreSQL and Temporal. olake-tui cuts out the middleman:

```
  Standard OLake Web Stack              olake-tui
  ========================              =========

  ┌───────────┐                         ┌─────────────┐
  │  Browser   │                         │  olake-tui   │  ← You are here
  └─────┬─────┘                         └──────┬───────┘
        │ HTTP                                  │ Direct SQL + gRPC
  ┌─────▼─────┐                                 │
  │ BFF Server │                                 │
  └─────┬─────┘                                 │
        │ SQL + gRPC                            │
  ┌─────▼──────────┐  ┌──────────────┐   ┌─────▼──────────┐  ┌──────────────┐
  │  PostgreSQL     │  │   Temporal    │   │  PostgreSQL     │  │   Temporal    │
  │  (shared DB)    │  │  (workflows)  │   │  (shared DB)    │  │  (workflows)  │
  └────────────────┘  └──────┬───────┘   └────────────────┘  └──────┬───────┘
                             │                                       │
                      ┌──────▼───────┐                        ┌──────▼───────┐
                      │  OLake Worker │                        │  OLake Worker │
                      │  (sync/check) │                        │  (sync/check) │
                      └──────────────┘                        └──────────────┘
```

**What talks to what:**

| Component | Protocol | Purpose |
|-----------|----------|---------|
| olake-tui → PostgreSQL | SQL (pgx driver) | All CRUD: sources, destinations, jobs, users, settings |
| olake-tui → Temporal | gRPC | Workflow state, sync triggers, schedule management |
| Temporal → OLake Worker | gRPC | Dispatches discover/check/sync tasks |
| OLake Worker → Sources | Native protocols | Reads data from databases (pg, mysql, mongo, etc.) |
| OLake Worker → Destinations | SDK/API | Writes to Iceberg, S3/Parquet |

Because olake-tui reads and writes the **same database schema** as the BFF server, you can switch between the web UI and TUI freely. They are fully compatible — changes made in one are immediately visible in the other.

## Why TUI over Web UI?

The OLake web UI works well for teams with browser access and a running BFF server. But many real-world data engineering environments don't fit that model. Here are concrete scenarios where olake-tui shines:

### 1. Air-gapped and restricted networks

**Scenario:** Your production database runs in a network with no outbound internet and no browser access. Corporate policy blocks HTTP traffic to internal services from developer machines.

**With web UI:** You'd need to deploy the React frontend + BFF server inside the restricted network, configure TLS certificates, set up internal DNS — a full web deployment just to manage pipelines.

**With olake-tui:** SSH into any machine that can reach PostgreSQL and Temporal. Run the binary. Done. No web server, no certificates, no DNS.

```bash
# From your workstation, SSH into the production bastion
ssh bastion.prod.internal

# Run olake-tui directly — it only needs DB + Temporal connectivity
olake-tui --db-url "postgres://olake:pass@db.internal:5432/olake?sslmode=require"
```

### 2. SSH into production servers

**Scenario:** You're debugging a failed sync at 2 AM. You've SSH'd into the bastion host through two jump boxes. Opening a browser is not an option.

**With web UI:** Port-forward 3 services (frontend, BFF, maybe even the DB), open a browser, navigate to the job, check logs. That's 5 minutes of setup before you even see the problem.

**With olake-tui:** One command, instant access to job status and logs.

```bash
# Through your jump box chain
ssh -J jump1,jump2 bastion

# Immediately see what's going on
olake-tui --db-url "$OLAKE_DB_URL" --temporal-host temporal:7233
# Press 1 → see jobs → select failed job → press l → read logs
```

### 3. CI/CD pipeline automation

**Scenario:** Your team uses GitOps. When infrastructure is provisioned, you need to bootstrap the OLake database schema and create an admin user — without any interactive UI.

**With web UI:** Spin up the signup-init container, wait for it to initialize, then tear it down. Extra container image, extra complexity.

**With olake-tui:** Use `--migrate-only` in your init container or CI step.

```yaml
# In your Kubernetes Job or CI pipeline
- name: bootstrap-olake
  image: teampaprika/olake-tui
  command:
    - olake-tui
    - --db-url=postgres://olake:pass@postgres:5432/olake?sslmode=disable
    - --migrate-only
    - --admin-user=admin
    - --admin-pass=$ADMIN_PASSWORD
```

### 4. Keyboard-driven speed

**Scenario:** You manage 50 sync jobs. Every morning you check status, re-trigger failed ones, and review logs. In the web UI, that's click → wait → scroll → click → wait, repeated 50 times.

**With olake-tui:** Navigate with `j`/`k`, trigger syncs with `s`, check logs with `l`. You can review all 50 jobs in under a minute without touching a mouse.

```
Keyboard workflow (< 30 seconds):
  1 → Jobs view
  j/k → scroll through jobs
  s → trigger sync on selected job
  l → view logs
  Esc → back
  repeat
```

### 5. Lower resource footprint

**Scenario:** You're running OLake on a small VM or Raspberry Pi. Every megabyte of RAM matters.

| Component | Memory | Disk |
|-----------|--------|------|
| React Frontend | ~100MB | ~50MB (node_modules) |
| BFF Server (Node.js) | ~150MB | ~200MB (dependencies) |
| signup-init container | ~80MB | ~100MB |
| **olake-tui** | **~15MB** | **~10MB (single binary)** |

olake-tui replaces all three web components with a single 10MB binary using ~15MB of RAM.

### 6. Scriptable operations

**Scenario:** You want to run migrations as part of an automated deployment script, not interactively.

```bash
#!/bin/bash
# deploy-olake.sh — run after infrastructure is up

# Wait for PostgreSQL
until pg_isready -h localhost -p 5432; do sleep 1; done

# Wait for Temporal
until temporal operator cluster health; do sleep 1; done

# Bootstrap schema + admin user
olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --migrate-only \
  --admin-user admin \
  --admin-pass "$ADMIN_PASS"

echo "OLake is ready."
```

## Feature comparison: Web UI vs TUI

| Feature | Web UI | olake-tui |
|---------|--------|-----------|
| Create/edit/delete sources | ✅ | ✅ |
| Test source connections | ✅ | ✅ |
| Create/edit/delete destinations | ✅ | ✅ |
| Create jobs with stream selection | ✅ | ✅ |
| Full refresh sync | ✅ | ✅ |
| CDC (Change Data Capture) sync | ✅ | ✅ |
| Trigger manual sync | ✅ | ✅ |
| Cancel running jobs | ✅ | ✅ |
| View job logs (paginated) | ✅ | ✅ |
| Settings management | ✅ | ✅ |
| User authentication | ✅ | ✅ |
| Database schema migration | Via signup-init | Built-in (`--migrate`) |
| Non-interactive mode (CI/CD) | ❌ | ✅ (`--migrate-only`) |
| Works over SSH | ❌ | ✅ |
| Works without a browser | ❌ | ✅ |
| Works in air-gapped networks | Partial (needs deployment) | ✅ (single binary) |
| Keyboard-only navigation | Partial | ✅ |
| Resource footprint | ~330MB (3 services) | ~15MB (1 binary) |

## Supported connectors

olake-tui supports all OLake connectors:

**Sources:**
- PostgreSQL
- MySQL
- MongoDB
- Oracle
- MSSQL
- DB2
- Kafka
- S3

**Destinations:**
- Apache Iceberg
- Amazon S3 (Parquet)

## What you need

Before using olake-tui, you need three infrastructure components running:

| Component | Why | Default address |
|-----------|-----|-----------------|
| **PostgreSQL 13+** | Stores all OLake data: sources, destinations, jobs, users, settings | `localhost:5432` |
| **Temporal Server** | Orchestrates sync workflows — discover, check, sync | `localhost:7233` |
| **OLake Worker** | Executes the actual data movement between sources and destinations | Connected via Temporal |

The included `docker-compose.yml` starts all three (plus Elasticsearch for Temporal visibility) with a single command. See the [Quick Start]({{< relref "quick-start" >}}) for the full walkthrough.

{{< callout type="info" >}}
**Already running OLake with the web UI?** olake-tui can connect to your existing database. No migration needed — just point it at the same PostgreSQL instance and log in with your existing credentials.
{{< /callout >}}

## Troubleshooting

### "I already have OLake running — will olake-tui conflict?"

No. olake-tui uses the same database schema as the BFF server. You can run both simultaneously. Changes made in one are immediately visible in the other.

### "Do I need to install Temporal separately?"

If you use the included `docker-compose.yml`, Temporal is included. If you're connecting to an existing OLake deployment, Temporal is already running — just point `--temporal-host` at it.

### "What Go version do I need?"

Go 1.22 or later. Check with `go version`. If you don't want to install Go, use the pre-built Docker image: `docker pull teampaprika/olake-tui`.

### "Can I use olake-tui with OLake Cloud?"

olake-tui is designed for self-hosted OLake deployments where you have direct access to PostgreSQL and Temporal. It does not work with managed/cloud OLake instances that don't expose these services directly.

### "My terminal looks garbled / colors are wrong"

olake-tui requires a terminal that supports 256 colors and Unicode. Most modern terminals work fine:
- ✅ iTerm2, Alacritty, kitty, WezTerm, Windows Terminal
- ✅ macOS Terminal.app (recent versions)
- ⚠️ Older PuTTY versions may need color settings adjusted
- ❌ Very old terminals without Unicode support

If you see garbled output, try setting `TERM=xterm-256color` before running olake-tui.

## Next steps

- **[Quick Start]({{< relref "quick-start" >}})** — Get running in 5 minutes with a full end-to-end walkthrough
- **[Installation]({{< relref "installation" >}})** — Build from source, Docker image, or connect to existing infrastructure
- **[OLake Documentation](https://olake.io/docs/)** — Learn about the OLake platform itself
