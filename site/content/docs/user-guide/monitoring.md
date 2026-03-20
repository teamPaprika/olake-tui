---
title: "Monitoring"
weight: 6
---

OLake TUI provides real-time monitoring through a dashboard stats bar, Temporal-backed job status, task history tables, and a paginated log viewer. This page explains every monitoring feature, how the data flows, and how to interpret what you see.

## Prerequisites

- OLake TUI is running and connected (see [Connecting](../connecting/))
- For real-time status and task history: **Temporal server must be running**
- Jobs must exist (see [Jobs](../jobs/))

---

## Dashboard Stats Bar

The stats bar sits at the top of the Jobs tab, giving you an at-a-glance view of pipeline health:

```
 ⬡ OLake   logged in as admin
 ╭────────────────────────────────────────────────────────────────────╮
 │ Sources: 2   Destinations: 1   Jobs: 5   Active: 3   Running: 1  │
 ╰────────────────────────────────────────────────────────────────────╯
```

**What each counter means:**

| Counter | Description | Where It Comes From |
|---------|-------------|-------------------|
| **Sources** | Total configured source connectors | Database count (includes all types: PG, MySQL, etc.) |
| **Destinations** | Total configured destinations | Database count |
| **Jobs** | Total jobs (all states) | Database count |
| **Active Jobs** | Jobs with scheduling enabled (`activate = true`) | Jobs where the Temporal schedule is active |
| **Running** | Jobs currently executing a sync | Jobs with `WORKFLOW_EXECUTION_STATUS_RUNNING` in Temporal |

**Visual highlight:** The "Active Jobs" and "Running" cards get a cyan border when their count is > 0, making active work immediately visible.

---

## Real-Time Job Status

### How It Works

Job statuses are **not** stored in the OLake database — they come directly from Temporal via gRPC queries:

```
┌─────────┐         ┌──────────┐         ┌──────────┐
│ OLake   │  gRPC   │ Temporal │         │ OLake    │
│ TUI     │────────>│ Frontend │────────>│ Worker   │
│         │<────────│          │<────────│          │
│ status  │ query   │ workflow │ execute │ sync     │
│ display │ result  │ state    │ report  │ logic    │
└─────────┘         └──────────┘         └──────────┘
```

1. TUI queries Temporal for each job's latest workflow execution status
2. Temporal responds with the current state (running, completed, failed, etc.)
3. The query has a **5-second timeout** — if Temporal doesn't respond in time, the status falls back to `Unknown`

### Status Icons and Meanings

| Status | Meaning | What You Should Do |
|--------|---------|-------------------|
| **Idle** | No active workflow. Waiting for the next scheduled trigger or manual sync. | Normal state between syncs. |
| **Running** | A sync workflow is currently executing. | Wait for completion, or press `l` to watch logs. |
| **Completed** | The last sync finished successfully. | Nothing — check the row count in task history. |
| **Failed** | The last sync encountered an error. | Press `l` to view logs and diagnose the failure. |
| **Canceled** | The last sync was manually canceled (via `c`). | Intentional. Trigger a new sync with `s` when ready. |
| **Paused** | The job schedule is paused. No automatic syncs will trigger. | Press `p` to resume when ready. |
| **Unknown** | Temporal query timed out or Temporal is disconnected. | Check Temporal connectivity. |

### Auto-Refresh

The job list refreshes statuses automatically when you navigate back to it. Press `r` to force a manual refresh at any time.

---

## Job Detail View

Press `Enter` on any job to see its detail view with task history:

```
 ┌─ Job: pg-to-iceberg-6h ─────────────────────────────────────────┐
 │                                                                  │
 │  Source:      production-pg (PostgreSQL)                         │
 │  Destination: lakehouse-iceberg (Iceberg)                       │
 │  Schedule:    0 */6 * * * (every 6 hours)                       │
 │  Status:      Completed                                         │
 │  Streams:     3 selected                                        │
 │                                                                  │
 │  Task History:                                                   │
 │  ┌────────────────────────────────────────────────────────────┐  │
 │  │ #   Status      Started              Duration   Rows      │  │
 │  │────────────────────────────────────────────────────────────│  │
 │  │ 5   ✓ Completed 2025-03-15 06:00:01  4m 32s     1,248,301│  │
 │  │ 4   ✓ Completed 2025-03-15 00:00:01  4m 18s     1,201,455│  │
 │  │ 3   ✗ Failed    2025-03-14 18:00:01  0m 12s     0        │  │
 │  │ 2   ✓ Completed 2025-03-14 12:00:02  4m 45s     1,305,112│  │
 │  │ 1   ✓ Completed 2025-03-14 06:00:01  4m 22s     1,190,877│  │
 │  └────────────────────────────────────────────────────────────┘  │
 │                                                                  │
 │  s:sync  c:cancel  l:logs  S:settings  p:pause  esc:back        │
 └──────────────────────────────────────────────────────────────────┘
```

### Task History Table

Each row represents one sync execution (a Temporal workflow run):

| Column | Description |
|--------|-------------|
| **#** | Sequential run number within the job |
| **Status** | Outcome: `Completed`, `Failed`, `Canceled`, `Running`, `Timed Out` |
| **Started** | Timestamp when the workflow started |
| **Duration** | Wall-clock time from start to completion |
| **Rows** | Total rows synced (0 for failures/cancellations) |

**Navigation within task history:**
- `↑`/`↓` or `j`/`k` — scroll through tasks
- `Enter` or `l` — open logs for the selected task
- `PgUp`/`PgDn` — scroll one page (for long histories)

The task list is **paginated** — initially loads the most recent 15 tasks. Scroll past the end to load more.

### Reading the History

A healthy job looks like consistent completion times and row counts:
```
 5   ✓ Completed  4m 32s   1,248,301
 4   ✓ Completed  4m 18s   1,201,455
 3   ✓ Completed  4m 45s   1,305,112
```

Warning signs:
- **Duration increasing rapidly** → Source table growing, might need to switch from full_refresh to CDC
- **Row count drops to 0 on a "Completed" run** → No new data (normal for incremental/CDC with no changes)
- **Alternating Failed/Completed** → Intermittent connectivity issue; check network stability
- **Failed with 0m 12s duration** → Fast failure, usually a connection or authentication error

---

## Log Viewer

Press `l` on a job (from the list or detail view) to open the log viewer for the most recent task. From the task history, select a specific task and press `Enter` or `l`.

```
 ┌─ Logs — Job 1 / Task abc-123 ───────────────────────────────────┐
 │ 42 log entries                                                   │
 │                                                                  │
 │ 2025-03-15 06:00:01  INFO   Starting sync for job               │
 │                              "pg-to-iceberg-6h"                  │
 │ 2025-03-15 06:00:02  INFO   Discovered 3 streams, 3 selected    │
 │ 2025-03-15 06:00:03  INFO   Syncing public.users (cdc)          │
 │ 2025-03-15 06:00:04  INFO   Starting initial snapshot for       │
 │                              public.users                        │
 │ 2025-03-15 06:01:15  INFO   Snapshot complete: 50,000 rows      │
 │ 2025-03-15 06:01:16  INFO   Switching to CDC mode               │
 │ 2025-03-15 06:01:20  INFO   Syncing public.orders (cdc)         │
 │ 2025-03-15 06:02:15  WARN   Slow query detected on              │
 │                              public.orders (>30s)                │
 │ 2025-03-15 06:03:10  INFO   Syncing public.products             │
 │                              (full_refresh)                      │
 │ 2025-03-15 06:04:32  INFO   Sync completed: 1,248,301 rows      │
 │                              in 4m 32s                           │
 │                                                                  │
 │ ↑↓/pgup/pgdn: scroll  •  p:older  •  n:newer  •  esc: back     │
 └──────────────────────────────────────────────────────────────────┘
```

### Color Coding

Log lines are color-coded by severity level:

| Level | Color | Meaning |
|-------|-------|---------|
| `INFO` | Blue/Cyan | Normal operation messages |
| `WARN` | Yellow | Non-critical warnings (slow queries, retries) |
| `ERROR` | Red | Errors that may have caused the sync to fail |
| `FATAL` | Red (bold) | Critical errors that terminated the workflow |
| `DEBUG` | Dim gray | Verbose debug info (only visible in debug mode) |

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Scroll one line |
| `PgUp` / `PgDn` | Scroll one page |
| `p` | Load **older** log entries (paginate backward) |
| `n` | Load **newer** log entries (paginate forward) |
| `Esc` | Close log viewer and return to previous screen |

### Pagination

Logs are fetched in batches (default: 200 entries per page). The viewer starts at the **most recent** entries and scrolls down. When you need older entries:

1. Scroll to the top of the current page
2. Press `p` to load the previous (older) batch
3. The viewer reloads with older entries

Similarly, press `n` to load the next (newer) batch. The footer indicates when more pages are available.

---

## Interpreting Common Log Patterns

### Successful Full Refresh Sync

```
INFO   Starting sync for job "pg-products"
INFO   Discovered 1 streams, 1 selected
INFO   Syncing public.products (full_refresh)
INFO   Reading all rows from public.products
INFO   Read 15,230 rows from public.products
INFO   Writing to destination: lakehouse-iceberg
INFO   Write complete: 15,230 rows
INFO   Sync completed: 15,230 rows in 1m 05s
```

Clean sequential flow: discover → read → write → done.

### Successful CDC Sync (First Run)

```
INFO   Starting sync for job "pg-to-iceberg-6h"
INFO   Syncing public.orders (cdc)
INFO   No existing replication slot found. Creating initial snapshot.
INFO   Starting initial snapshot for public.orders
INFO   Snapshot: 1,200,000 rows read in 3m 45s
INFO   Snapshot complete. Creating replication slot "olake_orders"
INFO   Switching to CDC mode. Reading from WAL position 0/1A3B4C5D.
INFO   CDC: 48,301 changes captured
INFO   Sync completed: 1,248,301 rows in 4m 32s
```

The first CDC sync always does a **full snapshot** before switching to change capture. Subsequent runs will be much faster (only changes).

### Successful CDC Sync (Subsequent Run)

```
INFO   Starting sync for job "pg-to-iceberg-6h"
INFO   Syncing public.orders (cdc)
INFO   Resuming CDC from WAL position 0/1A3B4C5D
INFO   CDC: 12,455 changes captured (8,201 inserts, 4,254 updates)
INFO   Sync completed: 12,455 rows in 0m 38s
```

Much faster — only incremental changes since the last sync.

### Failed Sync: Source Unreachable

```
INFO   Starting sync for job "pg-to-iceberg-6h"
ERROR  Failed to connect to source: dial tcp 10.0.1.50:5432: connection refused
ERROR  Sync failed after 0m 12s
```

The source database is down or unreachable from the worker. Check the source server and network.

### Failed Sync: Destination Write Error

```
INFO   Starting sync for job "pg-to-iceberg-6h"
INFO   Read 1,248,301 rows from source
ERROR  Failed to write to destination: Access Denied (s3:PutObject)
ERROR  Sync failed after 3m 50s
```

Data was read successfully but couldn't be written. Check destination credentials and IAM permissions.

### Warning: CDC Lag

```
WARN   CDC lag detected: 45 minutes behind source
WARN   Consider increasing sync frequency or worker resources
```

The CDC consumer is falling behind the source's write rate. This means your destination data is increasingly stale. Solutions:
- Increase sync frequency (e.g., from `0 */6 * * *` to `0 * * * *`)
- Scale worker resources (more CPU/memory)
- Reduce the number of tables in a single job

---

## Troubleshooting

### "No task history" / Empty Task List

**Symptoms:** Job detail shows no tasks, even though the job has been created.

**Causes:**
1. **No sync has been triggered yet** — The job exists but hasn't run. Press `s` to trigger a manual sync, or wait for the cron schedule.
2. **Temporal is not connected** — Task history comes from Temporal. Without it, the TUI can't fetch workflow executions.
   ```bash
   # Check Temporal connectivity
   nc -z localhost 7233 && echo "OK" || echo "UNREACHABLE"
   ```
3. **Temporal data was cleared** — If Temporal's persistence was reset (e.g., Docker volume deleted), historical task data is lost.

### Logs Are Empty

**Symptoms:** Log viewer opens but shows "0 log entries".

**Causes:**
1. **The task failed before producing logs** — Check if the Temporal workflow even started:
   - Go to the Temporal Web UI (`http://localhost:8080`)
   - Find the workflow by ID (it contains the job name)
   - Check the workflow event history
2. **Worker is not writing logs to the expected path** — Logs are read from the shared config directory (`/tmp/olake-config` by default). Verify the worker and TUI share this volume:
   ```bash
   docker compose exec olake-worker ls -la /tmp/olake-config/
   ```
3. **Log file was cleaned up** — If auto-cleanup is configured, old log files may have been removed.

### Status Shows "Unknown" for All Jobs

**Cause:** Temporal is unreachable. The TUI queries Temporal for each job's status, and the 5-second timeout is being hit for every query.

**Fix:**
1. Check Temporal is running:
   ```bash
   docker compose ps temporal
   docker compose logs temporal --tail 20
   ```
2. Verify the address:
   ```bash
   nc -z localhost 7233
   ```
3. Restart the TUI after fixing Temporal connectivity.

### Status Stuck on "Running" After Cancellation

**Cause:** The cancellation signal was sent but the worker hasn't acknowledged it yet. The cancel operation has a 30-second grace period.

**Fix:** Wait up to 30 seconds. If it's still running after that:
1. Check the worker logs for the workflow
2. The workflow may have already completed before the cancel signal arrived
3. Press `r` to refresh — the status should update

### Dashboard Counters Don't Match

**Symptoms:** "Active: 3" but you see 5 jobs listed.

**Explanation:** "Active" only counts jobs with `activate = true` (scheduling enabled). Jobs that are paused, manual-only, or have no cron schedule are not counted as active even though they appear in the list.

---

## Monitoring Best Practices

### Set Up Alerts Outside OLake TUI

OLake TUI is great for interactive monitoring but not for alerting. For production:

1. **Temporal Web UI** (`http://localhost:8080`) — browse workflows, schedules, and search by status
2. **Prometheus + Grafana** — Temporal exports metrics; set alerts on `temporal_workflow_failed_total`
3. **Periodic log review** — Check `ERROR` and `WARN` patterns weekly

### Establish Baselines

When a job first starts running, note:
- **Typical duration** for a full sync
- **Typical row count** for incremental/CDC syncs
- **Typical CDC lag** (if applicable)

Deviations from these baselines are your early warning system.

### Handling First-Run vs Steady-State

The first CDC sync is always slow (full snapshot). Don't panic:
- First run: 4+ hours for a large table (millions of rows)
- Subsequent runs: seconds to minutes (only changes)

After the initial snapshot, monitor that subsequent runs stay consistent.

---

## Keyboard Shortcuts Reference

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Job list | Open job detail + task history |
| `l` | Job list / detail / task | Open log viewer |
| `r` | Job list | Force refresh statuses |
| `s` | Job list / detail | Trigger manual sync |
| `c` | Job list / detail | Cancel running sync |
| `p` | Job list | Pause / resume schedule |
| `↑`/`k` | Task history / logs | Scroll up |
| `↓`/`j` | Task history / logs | Scroll down |
| `PgUp`/`PgDn` | Logs | Scroll one page |
| `p` | Log viewer | Load older entries |
| `n` | Log viewer | Load newer entries |
| `Esc` | Any sub-screen | Go back |
