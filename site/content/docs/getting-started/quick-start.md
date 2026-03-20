---
title: "Quick Start"
weight: 2
---

# Quick Start

This guide walks you through setting up olake-tui from scratch and running your first data sync. By the end, you'll have a PostgreSQL source syncing to an S3/Parquet destination — all managed from your terminal.

**Time required:** ~10 minutes (5 for setup, 5 for your first pipeline).

{{< callout type="info" >}}
For the full OLake platform quickstart (web UI, BFF server, etc.), see the [OLake Quickstart Guide](https://olake.io/docs/getting-started/quickstart/). This guide covers the TUI-only path.
{{< /callout >}}

## Prerequisites checklist

Before you start, make sure you have:

| Prerequisite | Check command | Minimum version |
|--------------|---------------|-----------------|
| **Docker** | `docker --version` | 20.10+ |
| **Docker Compose** | `docker compose version` | 2.0+ |
| **Go** | `go version` | 1.22+ |
| **Git** | `git --version` | Any recent version |
| **Terminal** | — | 256-color, Unicode support |

{{< callout type="warning" >}}
**Don't have Go installed?** You can skip the build step and use the Docker image instead. See Step 2b below.
{{< /callout >}}

If you're unsure about your terminal's capabilities, run this quick test:

```bash
echo -e "\033[38;5;208m■ Color test\033[0m  ╭──╮ Box test  ✓ Unicode test"
```

You should see an orange square, box-drawing characters, and a checkmark. If any are garbled, consider upgrading your terminal emulator.

## Step 1: Start infrastructure

Clone the repository and start the required services:

```bash
git clone https://github.com/teamPaprika/olake-tui.git
cd olake-tui
docker compose up -d
```

**Expected output:**

```
[+] Running 5/5
 ✔ Network olake-tui_olake-network  Created           0.0s
 ✔ Volume "olake-tui_temporal-postgresql-data"  Created  0.0s
 ✔ Container temporal-postgresql     Started           0.3s
 ✔ Container temporal-elasticsearch  Started           0.4s
 ✔ Container temporal                Started           0.8s
 ✔ Container olake-worker            Started           1.0s
```

This starts four services:

| Service | Container name | Port | Purpose |
|---------|---------------|------|---------|
| PostgreSQL 13 | `temporal-postgresql` | `5432` | Shared database for Temporal + OLake data |
| Elasticsearch 7.17 | `temporal-elasticsearch` | `9200` | Temporal visibility backend |
| Temporal Server | `temporal` | `7233` | Workflow orchestration (gRPC) |
| OLake Worker | `olake-worker` | — | Executes sync/discover/check workflows |

**Verify everything is healthy:**

```bash
docker compose ps
```

```
NAME                        STATUS              PORTS
temporal-postgresql         running (healthy)   0.0.0.0:5432->5432/tcp
temporal-elasticsearch      running (healthy)   0.0.0.0:9200->9200/tcp
temporal                    running             0.0.0.0:7233->7233/tcp
olake-worker                running             
```

{{< callout type="warning" >}}
**Wait for PostgreSQL and Elasticsearch to show `healthy` before proceeding.** Temporal depends on both, and olake-tui needs Temporal. This typically takes 15-30 seconds.
{{< /callout >}}

## Step 2a: Build olake-tui from source

```bash
make build
```

**Expected output:**

```
go build -ldflags "-X main.version=v0.2.0-direct" -o bin/olake-tui ./cmd/olake-tui/
```

Verify the binary:

```bash
bin/olake-tui --version
```

```
olake-tui v0.2.0-direct
```

## Step 2b: Use the Docker image (alternative)

If you don't have Go installed, pull the pre-built image:

```bash
docker pull teampaprika/olake-tui
```

For the remaining steps, replace `bin/olake-tui` with:

```bash
docker run -it --rm --network host teampaprika/olake-tui
```

## Step 3: First run with `--migrate`

On the very first run, olake-tui needs to create its database tables and seed an admin user. The `--migrate` flag handles both, then launches the TUI:

```bash
bin/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233 \
  --migrate \
  --admin-user admin \
  --admin-pass changeme
```

**Expected output (before the TUI takes over the screen):**

```
Running database migration...
✓ Schema migration complete
Seeding admin user (admin)...
✓ Admin user ready
```

After these lines, olake-tui enters fullscreen mode (alternate screen buffer) and shows the login screen.

{{< callout type="info" >}}
**What `--migrate` does behind the scenes:**
1. Creates OLake tables in the PostgreSQL database (`sources`, `destinations`, `jobs`, `users`, `settings`, etc.)
2. Creates an admin user with the email/password you specified
3. Starts the interactive TUI

On subsequent runs, you can omit `--migrate` — the tables already exist.
{{< /callout >}}

## Step 4: Log in

The TUI shows a centered login form:

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                   │
│                                                                   │
│                        ⬡ OLake TUI                                │
│                                                                   │
│                   ┌────────────────────┐                          │
│                   │ username            │                          │
│                   │ admin               │                          │
│                   └────────────────────┘                          │
│                   ┌────────────────────┐                          │
│                   │ password            │                          │
│                   │ ••••••••            │                          │
│                   └────────────────────┘                          │
│                                                                   │
│                   [ Tab: next field ]                              │
│                   [ Enter: log in   ]                              │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

1. Type `admin` in the username field
2. Press `Tab` to move to the password field
3. Type `changeme`
4. Press `Enter` to log in

After successful login, you'll land on the **Dashboard**.

## Step 5: The Dashboard

The dashboard shows a summary of your OLake instance:

```
┌──────────────────────────────────────────────────────────────────┐
│  ⬡ OLake  logged in as admin                             v0.2.0 │
│                                                                   │
│  ╭──────────╮  ╭──────────────╮  ╭──────╮  ╭────────────╮  ╭───╮│
│  │ Sources: 0│  │ Destinations:0│  │Jobs:0│  │Active Jobs:0│  │⟳:0││
│  ╰──────────╯  ╰──────────────╯  ╰──────╯  ╰────────────╯  ╰───╯│
│                                                                   │
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings            │
│                                                                   │
│  (no jobs yet — press n to create one, or start with sources)     │
│                                                                   │
│  n:new  Enter:detail  S:settings  s:sync  c:cancel  l:logs       │
└──────────────────────────────────────────────────────────────────┘
```

Everything starts at zero. Let's create a source, a destination, and then a job that connects them.

## Step 6: Create a source

Press `2` to navigate to the **Sources** view, then press `n` to open the new source form.

**Step 6a: Choose a connector type**

```
┌──────────────────────────────────────────────────────────────────┐
│  Select Source Type                                               │
│                                                                   │
│  > PostgreSQL                                                     │
│    MySQL                                                          │
│    MongoDB                                                        │
│    Oracle                                                         │
│    MSSQL                                                          │
│    DB2                                                            │
│    Kafka                                                          │
│    S3                                                             │
│                                                                   │
│  ↑/↓: navigate  Enter: select  Esc: cancel                       │
└──────────────────────────────────────────────────────────────────┘
```

Use `j`/`k` or arrow keys to select, then press `Enter`.

**Step 6b: Fill in connection details**

For a PostgreSQL source, you'll see this form:

```
┌──────────────────────────────────────────────────────────────────┐
│  New Source: PostgreSQL                                           │
│                                                                   │
│  Name:     ┌──────────────────────────────────┐                  │
│            │ my-postgres                       │                  │
│            └──────────────────────────────────┘                  │
│  Host:     ┌──────────────────────────────────┐                  │
│            │ localhost                          │                  │
│            └──────────────────────────────────┘                  │
│  Port:     ┌──────────────────────────────────┐                  │
│            │ 5432                               │                  │
│            └──────────────────────────────────┘                  │
│  Username: ┌──────────────────────────────────┐                  │
│            │ myuser                             │                  │
│            └──────────────────────────────────┘                  │
│  Password: ┌──────────────────────────────────┐                  │
│            │ ••••••••                           │                  │
│            └──────────────────────────────────┘                  │
│  Database: ┌──────────────────────────────────┐                  │
│            │ mydb                               │                  │
│            └──────────────────────────────────┘                  │
│                                                                   │
│  Tab: next field  Enter: save  Esc: cancel                       │
└──────────────────────────────────────────────────────────────────┘
```

| Field | Description | Example |
|-------|-------------|---------|
| **Name** | Human-readable label shown in the TUI | `my-postgres` |
| **Host** | Database hostname or IP | `localhost`, `db.prod.internal` |
| **Port** | Database port | `5432` (PostgreSQL default) |
| **Username** | Database user with read access | `readonly_user` |
| **Password** | Database password (masked with `•`) | — |
| **Database** | Database name to connect to | `mydb` |

Press `Tab` to move between fields. Press `Enter` when done to save.

## Step 7: Test the connection

After saving the source, select it in the list and look for the test/check option. olake-tui sends a `check` workflow to Temporal, which the OLake Worker executes.

**Successful test:**

```
  ✓ Connection to my-postgres successful
```

**Failed test (example):**

```
  ✗ Connection failed: dial tcp 192.168.1.100:5432: connect: connection refused
```

If the test fails, select the source and edit it to fix the connection details.

## Step 8: Create a destination

Press `3` to navigate to **Destinations**, then press `n`.

**Choose a destination type:**

```
┌──────────────────────────────────────────────────────────────────┐
│  Select Destination Type                                          │
│                                                                   │
│  > Apache Iceberg                                                 │
│    Amazon S3 (Parquet)                                            │
│                                                                   │
│  ↑/↓: navigate  Enter: select  Esc: cancel                       │
└──────────────────────────────────────────────────────────────────┘
```

For S3/Parquet, fill in your bucket, region, and AWS credentials. For Apache Iceberg, provide the catalog configuration.

## Step 9: Create a job with the wizard

Press `1` to go to **Jobs**, then press `n` to launch the job creation wizard. The wizard has four steps:

**Step 1 of 4: Job configuration**

```
┌──────────────────────────────────────────────────────────────────┐
│  New Job — Step 1/4: Configuration                                │
│                                                                   │
│  Name:     ┌──────────────────────────────────┐                  │
│            │ my-first-sync                      │                  │
│            └──────────────────────────────────┘                  │
│                                                                   │
│  Enter: next step  Esc: cancel                                   │
└──────────────────────────────────────────────────────────────────┘
```

**Step 2 of 4: Select source**

```
┌──────────────────────────────────────────────────────────────────┐
│  New Job — Step 2/4: Select Source                                │
│                                                                   │
│  > my-postgres (PostgreSQL)                                       │
│                                                                   │
│  ↑/↓: navigate  Enter: select  Esc: back                         │
└──────────────────────────────────────────────────────────────────┘
```

**Step 3 of 4: Select destination**

```
┌──────────────────────────────────────────────────────────────────┐
│  New Job — Step 3/4: Select Destination                           │
│                                                                   │
│  > my-s3-bucket (Amazon S3 Parquet)                               │
│                                                                   │
│  ↑/↓: navigate  Enter: select  Esc: back                         │
└──────────────────────────────────────────────────────────────────┘
```

**Step 4 of 4: Select streams and sync mode**

After selecting a source, olake-tui triggers a `discover` workflow to fetch available tables/streams. You'll see a spinner while it runs:

```
  ⣾ Discovering streams from my-postgres...
```

Once discovery completes, you can select which streams (tables) to sync and configure the sync mode for each:

```
┌──────────────────────────────────────────────────────────────────┐
│  New Job — Step 4/4: Select Streams                               │
│                                                                   │
│  [x] public.users              Full Refresh                       │
│  [x] public.orders             CDC (Incremental)                  │
│  [ ] public.sessions           —                                  │
│  [x] analytics.events          CDC (Incremental)                  │
│  [ ] analytics.page_views      —                                  │
│                                                                   │
│  Space: toggle  m: change sync mode  Enter: create job  Esc: back │
└──────────────────────────────────────────────────────────────────┘
```

- Press `Space` to toggle a stream on/off
- Press `m` to cycle the sync mode (Full Refresh ↔ CDC)
- Press `Enter` to create the job

## Step 10: Trigger a sync

Back on the Jobs list, your new job appears. Select it and press `s` to trigger a sync:

```
┌──────────────────────────────────────────────────────────────────┐
│  [1] Jobs  [2] Sources  [3] Destinations  [4] Settings            │
│                                                                   │
│  ⟳  1  my-first-sync   my-postgres  my-s3-bucket  running    ●   │
│                                                                   │
│  n:new  Enter:detail  S:settings  s:sync  c:cancel  l:logs       │
│                                                                   │
│  ┌────────────────────────────────────────────┐                   │
│  │  ✓ Sync triggered for "my-first-sync"      │   ← toast        │
│  └────────────────────────────────────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
```

A toast notification confirms the sync was triggered. The job status changes to `running` (⟳).

## Step 11: View logs

Press `l` on the running (or completed) job to open the log viewer:

```
┌──────────────────────────────────────────────────────────────────┐
│  Logs: my-first-sync                                    Page 1/3 │
│                                                                   │
│  2024-01-15 14:23:01  Starting sync workflow                      │
│  2024-01-15 14:23:02  Connecting to source: my-postgres           │
│  2024-01-15 14:23:03  Discovered 3 streams                        │
│  2024-01-15 14:23:04  Syncing public.users (full refresh)         │
│  2024-01-15 14:23:05  Wrote 1,247 records to s3://bucket/users/   │
│  2024-01-15 14:23:06  Syncing public.orders (CDC)                 │
│  2024-01-15 14:23:10  Wrote 15,832 records to s3://bucket/orders/ │
│  2024-01-15 14:23:11  Sync completed successfully                 │
│                                                                   │
│  ←/→: prev/next page  Esc: back                                  │
└──────────────────────────────────────────────────────────────────┘
```

Logs are paginated — use `←`/`→` or `h`/`l` to navigate pages.

## Key bindings reference

| Key | Context | Action |
|-----|---------|--------|
| `1` | Any | Switch to Jobs view |
| `2` | Any | Switch to Sources view |
| `3` | Any | Switch to Destinations view |
| `4` | Any | Switch to Settings view |
| `n` | List views | Create new item |
| `Enter` | List views | View details / select |
| `s` | Jobs | Trigger sync |
| `c` | Jobs | Cancel running job |
| `l` | Jobs | View logs |
| `S` | Any | Open Settings |
| `j` / `↓` | Lists | Move cursor down |
| `k` / `↑` | Lists | Move cursor up |
| `Tab` | Forms | Next field |
| `Shift+Tab` | Forms | Previous field |
| `Space` | Stream selection | Toggle stream on/off |
| `Esc` | Any | Go back / cancel |
| `q` / `Ctrl+C` | Any | Quit olake-tui |

## Optional: Temporal Web UI

For debugging workflow execution, start the Temporal web UI:

```bash
docker compose --profile debug up -d
```

Then open [http://localhost:8081](http://localhost:8081) in your browser. This is useful for inspecting workflow history, seeing retry attempts, and diagnosing sync failures at the orchestration level.

## What's next

- **[Installation]({{< relref "installation" >}})** — Alternative installation methods, Docker image, Kubernetes setup
- **[User Guide]({{< relref "../user-guide" >}})** — Deep dives into sources, destinations, jobs, and settings
- **[OLake Documentation](https://olake.io/docs/)** — Learn about OLake connectors and sync modes

## Troubleshooting

### "Error connecting to database: dial tcp localhost:5432: connect: connection refused"

PostgreSQL isn't ready yet. Check its status:

```bash
docker compose ps temporal-postgresql
# Should show "running (healthy)"

# If it's still starting, wait and retry:
docker compose logs temporal-postgresql --tail 20
```

**Fix:** Wait 15-30 seconds after `docker compose up` for PostgreSQL to initialize. You can also check readiness explicitly:

```bash
# This will block until PostgreSQL is ready
until docker compose exec temporal-postgresql pg_isready -U temporal; do
  echo "Waiting for PostgreSQL..."
  sleep 2
done
echo "PostgreSQL is ready!"
```

### "Error connecting to database: password authentication failed"

The default credentials in `docker-compose.yml` are `temporal`/`temporal`. Make sure your `--db-url` matches:

```bash
--db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable"
```

If you changed the PostgreSQL password in your compose file, update the connection string accordingly.

### "Temporal connection failed" / workflows don't start

Temporal takes longer to start than PostgreSQL because it depends on both PostgreSQL and Elasticsearch.

```bash
# Check Temporal logs
docker compose logs temporal --tail 30

# Verify Temporal is accepting gRPC connections
docker compose exec temporal temporal operator cluster health
```

**Fix:** Wait 30-60 seconds after `docker compose up`. Temporal needs Elasticsearch to be healthy first.

### "Login failed" / "Invalid credentials"

If you used `--migrate` on first run, the admin credentials are what you passed to `--admin-user` and `--admin-pass`. The default in this guide is `admin` / `changeme`.

If you forgot the credentials, re-run with `--migrate` — it will re-seed the admin user:

```bash
bin/olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --migrate \
  --admin-user admin \
  --admin-pass newpassword
```

### "Discover failed" / no streams appear

The discover workflow runs on the OLake Worker, not in olake-tui itself. Check that the worker is running:

```bash
docker compose ps olake-worker
docker compose logs olake-worker --tail 30
```

Also verify that your source connection details are correct and the source database is accessible from within the Docker network.

### Terminal display issues

If the TUI looks garbled or misaligned:

```bash
# Ensure your TERM is set correctly
echo $TERM
# Should be xterm-256color, screen-256color, or similar

# If not, set it:
export TERM=xterm-256color

# Then re-run olake-tui
```

Also ensure your terminal window is at least 80 columns wide and 24 rows tall. Resize if needed — the TUI adapts to terminal size changes dynamically.
