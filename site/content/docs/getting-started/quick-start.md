---
title: "Quick Start"
weight: 2
---

# Quick Start

Get olake-tui running in under 5 minutes. This guide assumes you have Docker and Docker Compose installed.

{{< callout type="info" >}}
For the full OLake platform quickstart (web UI, BFF server, etc.), see the [OLake Quickstart Guide](https://olake.io/docs/getting-started/quickstart/).
{{< /callout >}}

## 1. Clone the repository

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
```

## 2. Start the infrastructure

The included `docker-compose.yml` runs only what olake-tui needs — no BFF server, no web frontend:

```bash
docker compose up -d
```

This starts four services:

| Service | Port | Purpose |
|---------|------|---------|
| PostgreSQL | 5432 | Shared database for Temporal and OLake |
| Temporal | 7233 | Workflow orchestration (gRPC) |
| Elasticsearch | 9200 | Temporal visibility backend |
| OLake Worker | — | Executes sync/discover/check workflows |

Wait a few seconds for all services to become healthy:

```bash
docker compose ps
```

All containers should show `running` (or `healthy` if health checks are configured).

## 3. Build olake-tui

```bash
make build
```

This produces the binary at `bin/olake-tui`. Alternatively, install it to your `$GOPATH/bin`:

```bash
make install
```

{{< callout type="info" >}}
Requires Go 1.22 or later. See [Installation]({{< relref "installation" >}}) for detailed build instructions.
{{< /callout >}}

## 4. Bootstrap the database

On first run, use `--migrate` to create OLake tables and an admin user:

```bash
bin/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233 \
  --migrate \
  --admin-user admin \
  --admin-pass changeme
```

This does two things in sequence:
1. Creates the OLake schema tables in the PostgreSQL database
2. Creates an admin user with the specified credentials
3. Launches the TUI

If you only want to run the migration without starting the TUI (useful for CI/CD), use `--migrate-only`:

```bash
bin/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --migrate-only \
  --admin-user admin \
  --admin-pass changeme
```

## 5. Log in

When the TUI starts, you'll see a login screen. Enter the credentials you set in the previous step:

- **Email:** `admin`
- **Password:** `changeme`

After login, you'll land on the **Dashboard** showing counts for sources, destinations, and jobs.

## 6. Create a source

Press `2` to navigate to the **Sources** view, then press `n` to create a new source.

You'll be prompted for:
1. **Source name** — a human-readable label (e.g., `my-postgres`)
2. **Source type** — select from available connectors (MongoDB, PostgreSQL, MySQL)
3. **Connection details** — host, port, database, credentials

After filling in the form, you can **test the connection** before saving.

## 7. Create a destination

Press `3` to navigate to **Destinations**, then `n` to create one.

Configure your output target (S3, Apache Iceberg, or local Parquet) with the required credentials and paths.

## 8. Create and run a job

Press `1` to go to **Jobs**, then `n` to create a new job:

1. **Select source** — pick from your configured sources
2. **Select destination** — pick an output target
3. **Select streams** — choose which tables/collections to sync
4. **Configure sync mode** — full refresh or CDC (change data capture) per stream
5. **Save** the job

Back on the job list, select your new job and press `s` to trigger a sync.

Press `l` on any job to view its logs.

## Key bindings reference

| Key | Action |
|-----|--------|
| `1` | Jobs view |
| `2` | Sources view |
| `3` | Destinations view |
| `4` | Settings view |
| `n` | Create new item |
| `Enter` | View details |
| `s` | Trigger sync (jobs) |
| `c` | Cancel running job |
| `l` | View logs |
| `S` | Settings |
| `q` / `Ctrl+C` | Quit |

## Optional: Temporal Web UI

For debugging workflows, start the Temporal web UI:

```bash
docker compose --profile debug up -d
```

Then open [http://localhost:8081](http://localhost:8081) in your browser.

## Next steps

- [Installation]({{< relref "installation" >}}) — Alternative installation methods
- [User Guide]({{< relref "../user-guide" >}}) — Detailed feature documentation
