---
title: "Standalone (Docker Compose)"
weight: 1
---


The quickest way to run OLake with `olake-tui` is using the Docker Compose file
shipped at the repository root. It brings up all required infrastructure — no BFF
server needed.

## Architecture

The `docker-compose.yml` runs four services:

| Service            | Image                          | Purpose                                    |
|--------------------|--------------------------------|--------------------------------------------|
| **PostgreSQL**     | `postgres:13`                  | Shared database for Temporal and OLake     |
| **Elasticsearch**  | `elasticsearch:7.17.10`        | Temporal visibility store                  |
| **Temporal**       | `temporalio/auto-setup:1.22.3` | Workflow orchestration                     |
| **OLake Worker**   | `olakego/ui-worker:latest`     | Executes sync/discover/check workflows     |

`olake-tui` itself runs on your host machine — it connects directly to PostgreSQL
and Temporal over their exposed ports.

## Prerequisites

- Docker Engine ≥ 20.10 and Docker Compose v2
- `olake-tui` binary ([install instructions]({{< relref "/docs/getting-started/installation" >}}))
- Ports **5432**, **7233** available on the host

## Quick Start

### 1. Start Infrastructure

```bash
cd olake-tui
docker compose up -d
```

Wait for all services to become healthy:

```bash
docker compose ps
```

### 2. Run Migrations and Start TUI

```bash
olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233 \
  --migrate \
  --admin-user admin \
  --admin-pass changeme
```

The `--migrate` flag creates OLake tables in PostgreSQL and seeds the initial admin
user on first run. On subsequent launches you can omit `--migrate`, `--admin-user`,
and `--admin-pass`.

### 3. Verify

Once the TUI starts, press **S** to open the Sources view and confirm you can reach
the backend. If the worker container is running, the Temporal connection indicator
in the status bar will show **connected**.

## Temporal Web UI (Debug Profile)

The compose file includes an optional [Temporal Web UI](https://github.com/temporalio/ui)
behind a Docker Compose **profile**. To start it alongside the main services:

```bash
docker compose --profile debug up -d
```

This launches the Temporal UI on **http://localhost:8081**. Use it to inspect
workflows, view event histories, and debug stuck jobs.

To stop just the debug UI without touching other services:

```bash
docker compose --profile debug stop temporal-ui
```

## Private Registry / Air-Gapped Environments

All images respect the `CONTAINER_REGISTRY_BASE` environment variable. To pull from
an internal mirror instead of Docker Hub:

```bash
CONTAINER_REGISTRY_BASE=registry.internal.example.com docker compose up -d
```

See [Air-Gapped Deployment]({{< relref "air-gapped" >}}) for full offline instructions.

## Data Persistence

Two named volumes are created automatically:

- `temporal-postgresql-data` — PostgreSQL data directory
- `temporal-elasticsearch-data` — Elasticsearch indices

The OLake Worker also bind-mounts `./olake-data` on the host for connector config
files. Ensure this directory exists and is writable.

## Stopping and Cleanup

```bash
# Stop services (preserves data)
docker compose down

# Stop services AND delete volumes (full reset)
docker compose down -v
```

## Environment Variables

Instead of CLI flags you can export environment variables:

```bash
export OLAKE_DB_URL="postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"
export TEMPORAL_ADDRESS="localhost:7233"
export OLAKE_ADMIN_USER="admin"
export OLAKE_ADMIN_PASSWORD="changeme"

olake-tui --migrate
```

See [Configuration Reference]({{< relref "configuration" >}}) for the full list.
